package main

type DBData struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type UploadCommandRunner struct {
	Filepath    string
	Fullpath    string
	Sha256      string
	Sha1        string
	Md5         string
	Crc32       string
	Ahash       string
	Dhash       string
	ContentType string
}
