package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"log"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/klauspost/compress/zstd"
)

const version = "1.3.5"

var (
	dataDir     = flag.String("d", "./data", "Folder to store files")
	port        = flag.Int("p", 8080, "Port to listen on")
	compress    = flag.Bool("c", false, "Enable compression")
	level       = flag.Int("l", 3, "Compression level")
	dbFile      = flag.String("db", "./data/shortener.db", "SQLite database file to use for the url shortener")
	noSpeedtest = flag.Bool("disable-speedtest", false, "Disable speedtest")
	logging     = flag.Bool("log", false, "Enable logging")
)

var downloadSpeeds []float64
var db *sql.DB

func init() {
	flag.Parse()
	if *logging {
		logFile, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}

		log.SetOutput(logFile)
	}
}

func main() {
	fmt.Println("Starting")

	// Initialize the SQLite database
	var err error
	db, err = sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&mode=rwc", *dbFile))
	if err != nil {
		log.Fatal(err)
	}
	onStart()
	initDB()

	speed, err := testDownloadSpeed(10, 5*time.Second)
	if err != nil {
		log.Fatalf("Error testing download speed: %v", err)
	}
	downloadSpeeds = append(downloadSpeeds, speed)

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				speed, err := testDownloadSpeed(10, 5*time.Second)
				if err != nil {
					log.Fatalf("Error testing download speed: %v", err)
				}
				if err == nil {
					downloadSpeeds = append(downloadSpeeds, speed) // This causes a tiny memory leak. Too bad!
				}
			}
		}
	}()

	fmt.Println("Started")
	fmt.Println("Listening on port", *port)

	http.HandleFunc("/exists", handleExists)
	http.HandleFunc("/store", handleStore)
	http.HandleFunc("/get/", handleGet)
	http.HandleFunc("/get2/", handleGet2)
	http.HandleFunc("/stats", handleStats)
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/u/", handleU)
	http.HandleFunc("/shorten", handleShorten)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
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
func testDownloadSpeed(concurrentConnections int, testDuration time.Duration) (float64, error) {

	if *noSpeedtest {
		return 0, nil
	}

	var totalBytes int64
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, concurrentConnections)

	start := time.Now()
	for i := 0; i < concurrentConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get("https://speed.080609.xyz/downloading")
			if err != nil {
				errChan <- err
				return
			}
			defer resp.Body.Close()

			n, err := io.CopyN(io.Discard, resp.Body, math.MaxInt64)
			if err != nil && err != io.EOF {
				errChan <- err
				return
			}

			mu.Lock()
			totalBytes += n
			mu.Unlock()
		}()
	}

	// Wait for the specified test duration and then stop the test
	time.Sleep(testDuration)
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return 0, err
		}
	}

	duration := time.Since(start).Seconds()
	speed := float64(totalBytes) / duration // Speed in bytes per second

	return speed, nil
}
func initDB() {
	// Create table if it does not exist
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		hits INTEGER
	)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
}

func handleExists(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	_, err := os.Stat(request.ID)
	if err == nil {
		response := map[string]interface{}{
			"success": true,
			"error":   err,
			"id":      request.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	} else {
		response := map[string]interface{}{
			"success": false,
			"error":   err,
			"id":      request.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
}

func handleStore(w http.ResponseWriter, r *http.Request) {
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
}

func handleGet(w http.ResponseWriter, r *http.Request) {
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
}

func handleGet2(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}

	var filename string

	// Parse the query parameters
	params := r.URL.Query()
	hash := params.Get("h")
	ext := params.Get("e")
	filenameDown := params.Get("f")

	// If no hash is provided, default to '0'
	if hash == "" {
		hash = "0"
	}

	// If no extension is provided, default to 'bin'
	if ext == "" {
		ext = "bin"
	}

	// If no filename is provided, default to 'file.bin'
	if filenameDown == "" {
		filenameDown = "file.bin"
	}

	// Construct the filename
	filename = filepath.Join(*dataDir, hash)
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

	// Set the content type based on the file extension
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	// Set the content disposition to attachment with the provided filename
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filenameDown))

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
}

func handleStats(w http.ResponseWriter, r *http.Request) {
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
}

func handleShorten(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	hasher := crc32.NewIEEE()
	hasher.Write([]byte(request.URL))
	id := fmt.Sprintf("%x", hasher.Sum32())

	// Check if the Url is valid
	if len(request.URL) > 2048 {
		response := map[string]interface{}{
			"success": false,
			"error":   "URL is too long (2048 characters maximum)",
			"id":      "",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if the URL is already in the database
	var existingID string
	err := db.QueryRow("SELECT id FROM urls WHERE url = ?", request.URL).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Failed to check URL", http.StatusInternalServerError)
		return
	}

	if existingID != "" {
		// URL is already in the database, return the existing ID
		response := map[string]interface{}{
			"success": true,
			"error":   "",
			"id":      existingID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// URL is not in the database, insert it with hits set to  0
	_, err = db.Exec("INSERT INTO urls (id, url, hits) VALUES (?, ?,  0)", id, request.URL)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to store URL: %v", err),
			"id":      "",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"error":   "",
		"id":      id,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleU(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	id := r.URL.Path[len("/u/"):]

	var url string
	err := db.QueryRow("SELECT url FROM urls WHERE id = ?", id).Scan(&url)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Failed to retrieve URL", http.StatusInternalServerError)
		}
		return
	}

	// Increment the hits counter for the URL
	_, err = db.Exec("UPDATE urls SET hits = hits + 1 WHERE id = ?", id)
	if err != nil {
		log.Printf("Failed to increment hits for URL with id %s: %v", id, err)
	}

	http.Redirect(w, r, url, http.StatusFound)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}
	t := time.Now().UnixNano()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%d", t)))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}
