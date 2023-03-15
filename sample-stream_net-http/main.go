package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {

	uploadHandler := func(w http.ResponseWriter, req *http.Request) {
		reader, err := req.MultipartReader()

		if err != nil {
			fmt.Println("not multipart mime reader:", err)
		}

		for {
			part, err := reader.NextPart()
			if err != nil {
				if err == io.EOF {
					fmt.Println("Done")
					break
				} else {
					fmt.Println("Other type of err", err)
					return
				}
			}
			fmt.Println("FILENAME", part.FormName(), part.FileName(), part.Header.Get("Content-Type"))

			saving, err := os.Create(part.FileName())
			if err != nil {
				fmt.Println("not able to create file", err)
			}
			defer saving.Close()

			temp := bufio.NewWriter(saving)
			buffer := make([]byte, 1024*1024)
			for {
				read, err := part.Read(buffer)
				temp.Write(buffer[:read])
				if err == io.EOF {
					//fmt.Println("EOF", err, read)
					break
				}
				if err != nil {
					fmt.Println("Other type of error while saving", err)
				}
			}
			temp.Flush()
		}
		fmt.Fprintf(w, "body request response")
	} // helloHandler

	// curl -v -X POST --form file=@bnn.mkv http://localhost:3000/upload
	http.HandleFunc("/upload", uploadHandler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}
