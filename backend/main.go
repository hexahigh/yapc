package main

import (
	"bytes"
	"crypto"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"hash/crc64"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/webp"

	"github.com/hexahigh/go-lib/sniff"
	"github.com/hexahigh/yapc/backend/lib/hash"
	"github.com/peterbourgon/ff"
)

const version = "4.0.4"

var (
	dataDir              = flag.String("d", "./data", "Folder to store files")
	port                 = flag.Int("p", 8080, "Port to listen on")
	dbType               = flag.String("db", "sqlite", "Database type (sqlite or mysql)")
	dbPass               = flag.String("db:pass", "", "Database password (Unused for sqlite)")
	dbUser               = flag.String("db:user", "root", "Database user (Unused for sqlite)")
	dbHost               = flag.String("db:host", "localhost:3306", "Database host (Unused for sqlite)")
	dbDb                 = flag.String("db:db", "yapc", "Database name (Unused for sqlite)")
	dbFile               = flag.String("db:file", "./data/yapc.db", "SQLite database file")
	dbConns              = flag.Int("db:conns", 20, "Mysql database max open connections")
	fixDb                = flag.Bool("fixdb", false, "Fix the database")
	fixDb_dry            = flag.Bool("fixdb:dry", false, "Dry run fixdb")
	doResniff            = flag.Bool("resniff", false, "Resniff content-types")
	printLevel           = flag.Int("printlevel", 0, "Print/verbosity level (0-3)")
	disableUpload        = flag.Bool("disable:upload", false, "Disable uploading")
	disableShorten       = flag.Bool("disable:shorten", false, "Disable url shortening")
	commandToRunOnUpload = flag.String("run:upload", "", "Run a command on upload. View run.md for more info")
	waitForIt            = flag.Bool("wfi", false, "Wait for the database to be initialized")
	printLicense         = flag.Bool("l", false, "Print license")
	maxFileSize          = flag.Int64("maxfilesize", 1024*1024*1024*2, "Max file size in bytes")
)

var db *sql.DB
var logger *log.Logger

var (
	uploadCount   int64
	downloadCount int64
)

//go:embed LICENSE
var licenseFS embed.FS

func main() {

	image.RegisterFormat("webp", "RIFF????WEBPVP8 ", webp.Decode, webp.DecodeConfig)

	flag.Parse()
	ff.Parse(flag.CommandLine, os.Args[1:], ff.WithEnvVarPrefix("YAPC"))

	logger = log.New(os.Stdout, "", log.LstdFlags)

	if *printLicense {
		license, err := fs.ReadFile(licenseFS, "LICENSE")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(license))
		os.Exit(0)
	}

	fmt.Println("Starting")

	logLevelln(1, "Initializing database")
	// Initialize the SQLite database
	var err error
	switch *dbType {
	case "mysql":

		if *waitForIt {
			for {
				db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", *dbUser, *dbPass, *dbHost, *dbDb))
				if err != nil {
					log.Printf("Failed to connect to database: %v", err)
					time.Sleep(time.Second * 5)
					continue
				}
				break
			}
		} else {
			db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", *dbUser, *dbPass, *dbHost, *dbDb))
			if err != nil {
				log.Printf("Failed to connect to database: %v", err)
				os.Exit(1)
			}
		}
		db.SetConnMaxLifetime(time.Minute * 3)
		db.SetConnMaxIdleTime(time.Minute * 2)
		db.SetMaxOpenConns(*dbConns)
		db.SetMaxIdleConns(*dbConns)
	case "sqlite":
		db, err = sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&mode=rwc", *dbFile))
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Invalid database type: %s", *dbType)
		os.Exit(1)
	}

	defer db.Close()

	logLevelln(1, "Running initDB")
	initDB()

	if *fixDb {
		dbFixer()
	}

	if *doResniff {
		resniff()
	}

	logLevelln(1, "Running onStart")
	onStart()

	fmt.Println("Started")
	fmt.Println("Listening on port", *port)

	http.HandleFunc("/exists", handleExists)
	http.HandleFunc("/get/", handleGet)
	http.HandleFunc("/get2/", handleGet2)
	http.HandleFunc("/stats", handleStats)
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/u/", handleU)
	http.HandleFunc("/load", handleLoad)

	if !*disableUpload {
		http.HandleFunc("/store", handleStore)
	}

	if !*disableShorten {
		http.HandleFunc("/shorten", handleShorten)
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
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

	cleanHash := filepath.Clean(request.ID)
	if cleanHash != request.ID {
		http.Error(w, "Invalid hash", http.StatusBadRequest)
		logger.Println("An invalid hash was provided, perhaps someone tried to access files outside of the data folder." + request.ID)
		return
	}

	file := filepath.Join(*dataDir, cleanHash)
	_, err := os.Stat(file)
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
	atomic.AddInt64(&uploadCount, 1)
	defer atomic.AddInt64(&uploadCount, -1)

	type StoreResponse struct {
		SHA256 string `json:"sha256"`
		SHA1   string `json:"sha1"`
		MD5    string `json:"md5"`
		CRC32  string `json:"crc32"`
		AHash  string `json:"ahash"`
		DHash  string `json:"dhash"`
		Type   string `json:"type"`
	}

	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, *maxFileSize)

	file, _, err := r.FormFile("file")
	if err != nil {
		// Check if the error is due to the file size exceeding the limit
		if err.Error() == "http: request body too large" {
			http.Error(w, "File size too large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
		}
		return
	}
	defer file.Close()

	// Create a buffer to hold the file data
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	logLevelln(1, "Something was uploaded")

	// Create a wait group to wait for all hash computations to finish
	var wg sync.WaitGroup
	var mu sync.Mutex
	hashes := make(map[string]string)

	logLevelln(1, "Computing hashes")

	// Define a function to compute a hash and store it in the map
	computeHash := func(hashFunc crypto.Hash, hashKey string) {
		logLevelln(1, "Computing "+hashKey)
		defer wg.Done()
		hasher := hashFunc.New()
		hasher.Write(buf.Bytes())
		hash := hex.EncodeToString(hasher.Sum(nil))
		mu.Lock()
		hashes[hashKey] = hash
		mu.Unlock()
		logLevelln(1, "Computed "+hashKey)
	}

	// Compute SHA256, SHA1, MD5, and CRC32 hashes concurrently
	wg.Add(3)
	go computeHash(crypto.SHA256, "sha256")
	go computeHash(crypto.SHA1, "sha1")
	go computeHash(crypto.MD5, "md5")
	wg.Wait()

	// Compute CRC32 hash
	crc32Hasher := crc32.NewIEEE()
	crc32Hasher.Write(buf.Bytes())
	hashes["crc32"] = fmt.Sprintf("%x", crc32Hasher.Sum32())

	// Use SHA256 hash as the filename
	filename := *dataDir + "/" + hashes["sha256"]

	// Get the filetype based on magic number
	logLevelln(1, "Getting filetype")
	contentType := sniff.DetectContentType(buf.Bytes())

	if contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/gif" {
		logLevelln(1, "Detected image, computing Ahash and Dhash")
		// Decode the image from the buffer
		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			logger.Println("Failed to decode image", err)
		}

		// Choose the hash length
		hashLen := 32

		// Hash the image with Ahash
		ahashBytes, err := hash.Ahash(img, hashLen)
		if err != nil {
			logger.Println("Failed to generate Ahash", err)
		}

		// Hash the image with Dhash
		dhashBytes, err := hash.Dhash(img, hashLen)
		if err != nil {
			logger.Println("Failed to generate Dhash", err)
		}
		dHash := hex.EncodeToString(dhashBytes)
		aHash := hex.EncodeToString(ahashBytes)

		hashes["dhash"] = dHash
		hashes["ahash"] = aHash

	}

	absolutePath, err := filepath.Abs(filename)
	if err != nil {
		logLevelln(0, "Failed to get absolute path")
	}

	// Run the onupload command
	args := UploadCommandRunner{
		Filepath:    filename,
		Fullpath:    absolutePath,
		Sha256:      hashes["sha256"],
		Sha1:        hashes["sha1"],
		Md5:         hashes["md5"],
		Crc32:       hashes["crc32"],
		Ahash:       hashes["ahash"],
		Dhash:       hashes["dhash"],
		ContentType: contentType,
	}

	go runOnUpload(args)

	// Check if file already exists
	_, err = os.Stat(filename)
	if err == nil {
		// File already exists, return the JSON object with all hashes
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := StoreResponse{
			SHA256: hashes["sha256"],
			SHA1:   hashes["sha1"],
			MD5:    hashes["md5"],
			CRC32:  hashes["crc32"],
			AHash:  hashes["ahash"],
			DHash:  hashes["dhash"],
			Type:   contentType,
		}
		json.NewEncoder(w).Encode(response)

		return
	}

	logLevelln(1, "Saving file")

	newFile, err := os.Create(filename)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	// Write the file data to the new file
	if _, err := buf.WriteTo(newFile); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	logLevelln(1, "Storing hashes in database")

	// Write the hashes and the current Unix time to the "data" table in the database
	_, err = db.Exec(`INSERT INTO data (id, sha256, sha1, md5, crc32, ahash, dhash, type, uploaded) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hashes["sha256"], hashes["sha256"], hashes["sha1"], hashes["md5"], hashes["crc32"], hashes["ahash"], hashes["dhash"], contentType, time.Now().Unix())
	if err != nil {
		http.Error(w, "Failed to store hashes in database", http.StatusInternalServerError)
		return
	}

	// Set the content type to application/json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := StoreResponse{
		SHA256: hashes["sha256"],
		SHA1:   hashes["sha1"],
		MD5:    hashes["md5"],
		CRC32:  hashes["crc32"],
		AHash:  hashes["ahash"],
		DHash:  hashes["dhash"],
		Type:   contentType,
	}

	json.NewEncoder(w).Encode(response)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&downloadCount, 1)
	defer atomic.AddInt64(&downloadCount, -1)

	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}
	hash := r.URL.Path[len("/get/"):]

	cleanHash := filepath.Clean(hash)
	if cleanHash != hash {
		http.Error(w, "Invalid hash", http.StatusBadRequest)
		logger.Println("An invalid hash was provided, perhaps someone tried to access files outside of the data folder." + hash)
		return
	}

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
}

func handleGet2(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&downloadCount, 1)
	defer atomic.AddInt64(&downloadCount, -1)

	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}

	var filename string

	type Params struct {
		Hash        string
		Ext         string
		Filename    string
		ContentType string
	}

	// Parse the query parameters
	params := r.URL.Query()

	p := Params{
		Hash:        params.Get("h"),
		Ext:         params.Get("e"),
		Filename:    params.Get("f"),
		ContentType: params.Get("ct"),
	}

	// If no hash is provided, default to '0'
	if p.Hash == "" {
		http.Error(w, "No hash provided", http.StatusBadRequest)
		return
	}

	if strings.Contains(p.Hash, "..") {
		http.Error(w, "Invalid hash, did you try to access files outside of the data folder?", http.StatusBadRequest)
		return
	}

	// If no extension is provided, default to 'bin'
	if p.Ext == "" {
		p.Ext = "bin"
	}

	// If no filename is provided, default to 'file.bin'
	if p.Filename == "" {
		p.Filename = "file.bin"
	}

	// Query the database for the SHA256 hash associated with the provided hash
	var sha256Hash string
	err := db.QueryRow("SELECT sha256 FROM data WHERE sha256 = ? OR sha1 = ? OR md5 = ? OR crc32 = ?", p.Hash, p.Hash, p.Hash, p.Hash).Scan(&sha256Hash)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		} else {
			http.Error(w, "Failed to query database", http.StatusInternalServerError)
			return
		}
	}

	// Construct the filename using the SHA256 hash
	filename = filepath.Join(*dataDir, sha256Hash)

	fmt.Println("GET", r.URL.Path)
	fmt.Println("Attempting to get", filename)

	_, err = os.Stat(filename)
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

	// Get the file size
	fileInfo, err := os.Stat(filename)
	if err != nil {
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}
	fileSize := fileInfo.Size()

	w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))

	if p.ContentType == "" {
		// Set the content type based on the file extension
		contentType := mime.TypeByExtension(p.Ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", p.ContentType)
	}

	// Set the content disposition to attachment with the provided filename
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, p.Filename))

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Failed to send file", http.StatusInternalServerError)
		return
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

	cores := getCores()
	memInfo := getMem()

	response := map[string]interface{}{
		"uploadingDisabled":  *disableUpload,
		"shorteningDisabled": *disableShorten,
		"totalFiles":         len(files),
		"totalSize":          totalSize,
		"totalSpace":         totalSpace,
		"availableSpace":     availableSpace,
		"percentageUsed":     percentageUsed,
		"version":            version,
		"cores":              cores,
		"memory":             memInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func handleShorten(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var response struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		ID      string `json:"id"`
	}
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

	hasher := crc64.New(crc64.MakeTable(crc64.ISO))
	hasher.Write([]byte(request.URL))
	id := fmt.Sprintf("%x", hasher.Sum64())

	// Check if the Url is valid
	if len(request.URL) > 2048 {
		response.Success = false
		response.Error = "Url is too long (>2048 characters)"
		response.ID = ""
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if !isValidURL(request.URL) {
		response.Success = false
		response.Error = "Url is not valid"
		response.ID = ""
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
		response.Success = true
		response.Error = ""
		response.ID = existingID
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get current unix time
	uploadTime := time.Now().Unix()

	// URL is not in the database, insert it with hits set to  0
	_, err = db.Exec("INSERT INTO urls (id, url, hits, uploaded) VALUES (?, ?, 0, ?)", id, request.URL, uploadTime)
	if err != nil {
		response.Success = false
		response.Error = "Failed to insert URL into database: " + err.Error()
		response.ID = ""
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response.Error = ""
	response.Success = true
	response.ID = id
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

func handleLoad(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}

	response := map[string]interface{}{
		"uploads":   atomic.LoadInt64(&uploadCount),
		"downloads": atomic.LoadInt64(&downloadCount),
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func dbFixer() {
	logLevelln(0, "Cleaning database")

	logLevelln(0, "Looking for missing files...")
	// Query the database to get all IDs
	rows, err := db.Query("SELECT id FROM data")
	if err != nil {
		log.Fatalf("Failed to query database: %v", err)
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}

		// Construct the file path
		filePath := filepath.Join(*dataDir, id)

		// Check if the file exists
		_, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			// If the file does not exist, delete the entry from the database
			if !*fixDb_dry {
				_, err := db.Exec("DELETE FROM data WHERE id = ?", id)
				if err != nil {
					log.Printf("Failed to delete entry with ID %s: %v", id, err)
				}
			}
			log.Printf("Deleted entry with ID %s because the file does not exist", id)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating over rows: %v", err)
	}
}

func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func resniff() {
	// Query the database to get all file IDs
	rows, err := db.Query("SELECT id FROM data")
	if err != nil {
		log.Fatalf("Failed to query database: %v", err)
	}
	defer rows.Close()

	// Initialize a counter for the number of files processed
	var fileCount int

	// Get the total number of rows in the database
	var totalRows int
	err = db.QueryRow("SELECT COUNT(*) FROM data").Scan(&totalRows)
	if err != nil {
		log.Fatalf("Failed to get total row count: %v", err)
	}

	// Iterate over the rows
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}

		// Construct the file path
		filePath := filepath.Join(*dataDir, id)

		// Open the file
		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("Failed to open file %s: %v", filePath, err)
			continue
		}
		defer file.Close()

		// Read the first 1KB of the file
		buffer := make([]byte, 1024)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			log.Printf("Failed to read file %s: %v", filePath, err)
			continue
		}

		// Sniff the content type
		contentType := sniff.DetectContentType(buffer[:n])

		// Update the database with the new content type
		_, err = db.Exec("UPDATE data SET type = ? WHERE id = ?", contentType, id)
		if err != nil {
			log.Printf("Failed to update content type for file %s: %v", filePath, err)
			continue
		}

		// Increment the file counter
		fileCount++

		// Print the progress
		fmt.Printf("Processed file %d/%d\n", fileCount, totalRows)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating over rows: %v", err)
	}
}

func logLevelln(l int, s string) {
	if *printLevel >= l {
		logger.Println(s)
	}
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
func initDB() {
	// Create table if it does not exist
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id VARCHAR(255) PRIMARY KEY,
		url TEXT NOT NULL,
		hits INTEGER,
		uploaded INTEGER
	)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	// Create data table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS data (
		id VARCHAR(255) PRIMARY KEY,
		sha256 TEXT NOT NULL,
		sha1 TEXT NOT NULL,
		md5 TEXT NOT NULL,
		crc32 TEXT NOT NULL,
		ahash TEXT,
		dhash TEXT,
		type TEXT,
		uploaded INTEGER NOT NULL
	)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
}

func getCores() int {
	return runtime.NumCPU()
}

func getMem() runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats
}

func runOnUpload(args UploadCommandRunner) {
	if *commandToRunOnUpload == "" {
		return
	}

	modifiedCommand := *commandToRunOnUpload

	modifiedCommand = strings.Replace(modifiedCommand, "{%FILEPATH%}", args.Filepath, -1)
	modifiedCommand = strings.Replace(modifiedCommand, "{%FULLPATH%}", args.Fullpath, -1)
	modifiedCommand = strings.Replace(modifiedCommand, "{%SHA256%}", args.Sha256, -1)
	modifiedCommand = strings.Replace(modifiedCommand, "{%SHA1%}", args.Sha1, -1)
	modifiedCommand = strings.Replace(modifiedCommand, "{%MD5%}", args.Md5, -1)
	modifiedCommand = strings.Replace(modifiedCommand, "{%CRC32%}", args.Crc32, -1)
	modifiedCommand = strings.Replace(modifiedCommand, "{%AHASH%}", args.Ahash, -1)
	modifiedCommand = strings.Replace(modifiedCommand, "{%DHASH%}", args.Dhash, -1)
	modifiedCommand = strings.Replace(modifiedCommand, "{%CONTENTTYPE%}", args.ContentType, -1)

	// Split the command into executable and arguments
	parts := strings.Fields(modifiedCommand)
	cmd := exec.Command(parts[0], parts[1:]...)

	logLevelln(0, "Running command: "+modifiedCommand)

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Failed to execute command: %v\n", err)
	}
}
