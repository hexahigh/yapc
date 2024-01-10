package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/klauspost/compress/zstd"
)

const version = "1.1.3"

var (
	dataDir  = flag.String("d", "./data", "Folder to store files")
	port     = flag.Int("p", 8080, "Port to listen on")
	compress = flag.Bool("c", false, "Enable compression")
	level    = flag.Int("l", 3, "Compression level")
)

var downloadSpeeds []float64

func main() {
	flag.Parse()

	fmt.Println("Starting")

	onStart()

	speed, err := testDownloadSpeed()
	if err == nil {
		downloadSpeeds = append(downloadSpeeds, speed)
	}

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				speed, err := testDownloadSpeed()
				if err == nil {
					downloadSpeeds = append(downloadSpeeds, speed)
				}
			}
		}
	}()

	fmt.Println("Started")
	fmt.Println("Listening on port", *port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))

	http.HandleFunc("/store", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		hasher := crc32.NewIEEE()
		if _, err := io.Copy(hasher, file); err != nil {
			http.Error(w, "Failed to hash file", http.StatusInternalServerError)
			return
		}

		hash := fmt.Sprintf("%x", hasher.Sum32())
		filename := *dataDir + "/" + hash
		if *compress {
			filename += ".zst"
		}

		// Check if file already exists and return 200 and hash
		_, err = os.Stat(filename)
		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(hash))
			return
		}

		newFile, err := os.Create(filename)
		if err != nil {
			http.Error(w, "Failed to create file", http.StatusInternalServerError)
			return
		}
		defer newFile.Close()

		// Reset the file pointer to the beginning of the file
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			http.Error(w, "Failed to reset file pointer", http.StatusInternalServerError)
			return
		}

		if *compress {
			encoder, err := zstd.NewWriter(newFile, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(*level)))
			if err != nil {
				http.Error(w, "Failed to create encoder", http.StatusInternalServerError)
				return
			}
			defer encoder.Close()

			if _, err := io.Copy(encoder, file); err != nil {
				http.Error(w, "Failed to save file", http.StatusInternalServerError)
				return
			}
		} else {
			if _, err := io.Copy(newFile, file); err != nil {
				http.Error(w, "Failed to save file", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(hash))
	})

	http.HandleFunc("/get/", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		if r.Method == "OPTIONS" {
			return
		}
		hash := r.URL.Path[len("/get/"):]
		filename := filepath.Join(*dataDir, hash)
		if *compress {
			filename += ".zst"
		}

		fmt.Println("GET", r.URL.Path)
		fmt.Println("Attempting to get", filename)

		_, err := os.Stat(filename)
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}

		file, err := os.Open(filename)
		if err != nil {
			http.Error(w, "Failed to open file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		if *compress {
			decoder, err := zstd.NewReader(file)
			if err != nil {
				http.Error(w, "Failed to create decoder", http.StatusInternalServerError)
				return
			}
			defer decoder.Close()

			_, err = io.Copy(w, decoder.IOReadCloser())
			if err != nil {
				http.Error(w, "Failed to send file", http.StatusInternalServerError)
				return
			}
		} else {
			_, err = io.Copy(w, file)
			if err != nil {
				http.Error(w, "Failed to send file", http.StatusInternalServerError)
				return
			}
		}
	})

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Invalid request method, use GET", http.StatusMethodNotAllowed)
			return
		}

		files, err := fs.ReadDir(os.DirFS(*dataDir), ".")
		if err != nil {
			http.Error(w, "Failed to read directory", http.StatusInternalServerError)
			return
		}

		totalSize := int64(0)
		for _, file := range files {
			info, err := file.Info()
			if err != nil {
				http.Error(w, "Failed to get file info", http.StatusInternalServerError)
				return
			}
			totalSize += info.Size()
		}
		totalSpace, err := getTotalDiskSpace(*dataDir)
		if err != nil {
			http.Error(w, "Failed to get total disk space", http.StatusInternalServerError)
			return
		}

		availableSpace, err := getAvailableDiskSpace(*dataDir)
		if err != nil {
			http.Error(w, "Failed to get available disk space", http.StatusInternalServerError)
			return
		}

		percentageUsed := float64(totalSize) / float64(totalSpace) * 100

		var averageSpeed float64
		if len(downloadSpeeds) > 0 {
			var totalSpeed float64
			for _, speed := range downloadSpeeds {
				totalSpeed += speed
			}
			averageSpeed = totalSpeed / float64(len(downloadSpeeds))
		} else {
			averageSpeed = 0
		}

		response := map[string]interface{}{
			"totalFiles":        len(files),
			"totalSize":         totalSize,
			"totalSpace":        totalSpace,
			"availableSpace":    availableSpace,
			"percentageUsed":    percentageUsed,
			"compression":       *compress,
			"compression_level": *level,
			"version":           version,
			"averageSpeed":      averageSpeed,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func onStart() {
	// Check if data directory exists
	_, err := os.Stat(*dataDir)
	if os.IsNotExist(err) {
		// Create data directory
		err := os.Mkdir(*dataDir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
	// Check if there are uncompressed files and compression is on and vice versa
	files, err := os.ReadDir(*dataDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".zst" && *compress {
			// File is not compressed and compression is on, warn the user
			log.Printf("Warning: File %s is not compressed, but compression is enabled. You may want to compress this file.\n", file.Name())
		} else if filepath.Ext(file.Name()) == ".zst" && !*compress {
			// File is compressed and compression is off, warn the user
			log.Printf("Warning: File %s is compressed, but compression is disabled. You may want to decompress this file.\n", file.Name())
		}
	}
}

func getTotalDiskSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return stat.Blocks * uint64(stat.Bsize), nil
}
func getAvailableDiskSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}
func testDownloadSpeed() (float64, error) {
	start := time.Now()
	resp, err := http.Get("https://speed.080609.xyz/downloading")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, err
	}

	duration := time.Since(start).Seconds()
	speed := float64(n) / duration // Speed in bytes per second

	return speed, nil
}
