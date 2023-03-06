package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const uploadPath = "./uploads"

var maxUploadSize int64 = 1024 * 1024 // 1MB

// Progress is used to track the progress of a file upload.
// It implements the io.Writer interface so it can be passed
// to an io.TeeReader()
type Progress struct {
	TotalSize int64
	BytesRead int64
}

// Write is used to satisfy the io.Writer interface.
// Instead of writing somewhere, it simply aggregates
// the total bytes on each read
func (pr *Progress) Write(p []byte) (n int, err error) {
	n, err = len(p), nil
	pr.BytesRead += int64(n)
	pr.Print()
	return
}

// Print displays the current progress of the file upload
func (pr *Progress) Print() {
	if pr.BytesRead == pr.TotalSize {
		fmt.Println(" DONE!")
		return
	}

	fmt.Printf(" %d", pr.BytesRead*100/pr.TotalSize)
}

// web ui Handler serves the html file
func webUIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	http.ServeFile(w, r, "index.html")
}

// Healthz Handler for use in kubernetes
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	duration := time.Since(started)
	if duration.Seconds() > 10 {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("error: %v", duration.Seconds())))
		log.Fatal("health check takes too long: ", duration.String())
	} else {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
		log.Println("healthz check tooks: ", duration.String())
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 32 MB is the default used by FormFile
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// get a reference to the fileHeaders
	files := r.MultipartForm.File["file"]

	for _, fileHeader := range files {
		if fileHeader.Size > maxUploadSize {
			http.Error(w, fmt.Sprintf("The uploaded image is too big: %s. Please use an image less than 1MB in size", fileHeader.Filename), http.StatusBadRequest)
			continue
		}

		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer file.Close()

		buff := make([]byte, 512)
		_, err = file.Read(buff)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatal("Error reading file: ", fileHeader.Filename, " error is ", err)
			continue
		}

		filetype := http.DetectContentType(buff)
		if filetype != "image/jpeg" && filetype != "image/png" && filetype != "application/pdf" {
			http.Error(w, "The provided file format is not allowed. Please upload a JPEG, PNG or PDF file", http.StatusBadRequest)
			continue
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		f, err := os.Create(fmt.Sprintf("%s/%s_%d%s", uploadPath,
			fileHeader.Filename, //filepath.Base(fileHeader.Filename),
			time.Now().Unix()/10000,
			filepath.Ext(fileHeader.Filename)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		defer f.Close()

		pr := &Progress{
			TotalSize: fileHeader.Size,
		}
		log.Printf("File '%s' upload in progress (%%):", fileHeader.Filename)

		_, err = io.Copy(f, io.TeeReader(file, pr))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	fmt.Fprintf(w, "Upload successful")
}

func main() {

	err := os.MkdirAll(uploadPath, os.ModePerm)
	if err != nil {
		log.Fatal("Error ", err)
	}
	maxUploadSizeStr := os.Getenv("MAX_UPLOAD_SIZE")
	if maxUploadSizeStr != "" {
		maxUploadSize, err = strconv.ParseInt(maxUploadSizeStr, 10, 64)
		if err != nil {
			log.Fatal("Error ", err)
		}
	}

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "18899"
	}
	log.Println("severing on port", httpPort, "with max size of", maxUploadSize)

	mux := http.NewServeMux()
	mux.HandleFunc("/", webUIHandler)
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/upload", uploadHandler)

	if err = http.ListenAndServe(":4500", mux); err != nil {
		log.Fatal(err)
	}
}
