/*
Copyright © 2024 Simon Bråten <hexahigh0@gmail.com>
This file is part of yapc-cli
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"

	"github.com/hexahigh/yapc/cli/lib/config"
)

var (
	endpoint string
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload [file...]",
	Short: "Upload a file",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the endpoint from the config
		endpoint = config.GetString(*cfgFile, "Endpoint")
		// If args is empty then use filepicker
		if len(args) != 0 {
			for _, path := range args {
				if err := uploadFileOrDir(path); err != nil {
					fmt.Printf("Error uploading %s: %v\n", path, err)
				}
			}
		} else {
			fp := filepicker.New()
			fp.CurrentDirectory, _ = os.Getwd()
			fpm := fpModel{
				filepicker: fp,
			}
			tm, _ := tea.NewProgram(&fpm).Run()
			mm := tm.(fpModel)
			if mm.selectedFile != "" {
				uploadFile(mm.selectedFile)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

func uploadFileOrDir(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		// If it's a directory, upload all files within it
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, file := range files {
			if err := uploadFileOrDir(filepath.Join(path, file.Name())); err != nil {
				return err
			}
		}
	} else {
		// If it's a file, upload it
		uploadFile(path)
	}

	return nil
}

func uploadFile(path string) {
	// Get file size
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		return
	}
	fileSize := fileInfo.Size()

	bar := progressbar.DefaultBytes(fileSize, filepath.Base(path))

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", path, err)
		return
	}
	defer file.Close()

	// Create a buffer to store our request body
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create a form file field
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		fmt.Printf("Error creating form file: %v\n", err)
		return
	}

	// Create a custom writer that updates the progress bar
	customWriter := &progressWriter{
		writer: part,
		bar:    bar,
	}

	// Copy the file content to the custom writer
	_, err = io.Copy(customWriter, file)
	if err != nil {
		fmt.Printf("Error copying file content: %v\n", err)
		return
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		fmt.Printf("Error closing multipart writer: %v\n", err)
		return
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", endpoint+"/store", &requestBody)
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err)
		return
	}

	// Set the content type header
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending HTTP request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	type respJson struct {
		SHA256 string `json:"sha256"`
		SHA1   string `json:"sha1"`
		MD5    string `json:"md5"`
		CRC32  string `json:"crc32"`
	}

	// Declare a variable of type respJson
	var respData respJson

	// Decode the response into the respData variable
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return
	}

	// Now you can access the SHA256 field from respData
	fmt.Printf("Uploaded %s: %s\n", path, respData.SHA256)
}

type fpModel struct {
	filepicker   filepicker.Model
	selectedFile string
	quitting     bool
	err          error
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m fpModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m fpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case clearErrorMsg:
		m.err = nil
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	// Did the user select a file?
	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		// Get the path of the selected file.
		m.selectedFile = path

		m.quitting = true
		return m, tea.Quit
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.err = errors.New(path + " is not valid.")
		m.selectedFile = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}

func (m fpModel) View() string {
	if m.quitting {
		return ""
	}
	var s strings.Builder
	s.WriteString("\n  ")
	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
	} else if m.selectedFile == "" {
		s.WriteString("Pick a file:")
	} else {
		s.WriteString("Selected file: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
	}
	s.WriteString("\n\n" + m.filepicker.View() + "\n")
	return s.String()
}

type progressWriter struct {
	writer io.Writer
	bar    *progressbar.ProgressBar
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.writer.Write(p)
	if err != nil {
		return n, err
	}
	pw.bar.Add(n)
	return n, nil
}
