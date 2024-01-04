package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	dataDir = flag.String("d", "./data", "Folder to store files")
	port    = flag.Int("p", 8080, "Port to listen on")
)

func main() {
	flag.Parse()
	http.HandleFunc("/store", func(w http.ResponseWriter, r *http.Request) {
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

		hasher := sha256.New()
		if _, err := io.Copy(hasher, file); err != nil {
			http.Error(w, "Failed to hash file", http.StatusInternalServerError)
			return
		}

		hash := hex.EncodeToString(hasher.Sum(nil))
		filename := *dataDir + "/" + hash

		newFile, err := os.Create(filename)
		if err != nil {
			http.Error(w, "Failed to create file", http.StatusInternalServerError)
			return
		}
		defer newFile.Close()

		if _, err := io.Copy(newFile, file); err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(hash))
	})

	http.HandleFunc("/get/", func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Path[len("/get/"):]
		filename := filepath.Join(*dataDir, hash)

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

		_, err = io.Copy(w, file)
		if err != nil {
			http.Error(w, "Failed to send file", http.StatusInternalServerError)
			return
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
