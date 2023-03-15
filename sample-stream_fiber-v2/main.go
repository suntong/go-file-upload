package main

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {

	app := fiber.New()
	app.Server().StreamRequestBody = true
	app.Use(cors.New())

	app.Post("/upload", func(c *fiber.Ctx) error {
		// https://cs.opensource.google/go/go/+/refs/tags/go1.18:src/net/http/request.go;l=467
		v := c.Get("Content-Type")
		if v == "" {
			return nil
		}
		d, params, err := mime.ParseMediaType(v)
		if err != nil || !(d == "multipart/form-data" || d == "multipart/mixed") {
			return nil
		}
		boundary, ok := params["boundary"]
		if !ok {
			return nil
		}
		reader := multipart.NewReader(c.Context().RequestBodyStream(), boundary)
		for {
			part, err := reader.NextPart()
			if err != nil {
				if err == io.EOF {
					fmt.Println("EOF")
					break
				} else {
					fmt.Println("Other type of error", err)
					return nil
				}
			}
			fmt.Println("FILENAME", part.FormName(), part.FileName(), part.Header.Get("Content-Type"))
		}
		return nil
	})

	app.Listen(":3000")
}
