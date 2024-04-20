/*
Copyright © 2024 Simon Bråten <hexahigh0@gmail.com>
This file is part of yapc-cli
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	endpoint string
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload [file...]",
	Short: "Upload a file",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, path := range args {
			if err := uploadFileOrDir(path); err != nil {
				fmt.Printf("Error uploading %s: %v\n", path, err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	// Find and load the config file
	initConfig()

	// Get the endpoint from the config
	endpoint = viper.GetString("Endpoint")
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

	// Copy the file content to the form file field
	_, err = io.Copy(part, file)
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

type uploadModel struct {
	progress float64
}

func (m *uploadModel) Init() tea.Cmd {
	return nil
}

func (m *uploadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *uploadModel) View() string {
	return fmt.Sprintf("Uploading... %0.2f%%\n", m.progress*100)
}
