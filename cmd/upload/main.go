package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const uploadURL = "https://localhost:8083/upload"

func Close(body io.ReadCloser) {
	if body != nil {
		_ = body.Close()
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: upload <filename(s)...>")
	}
	for _, filename := range os.Args[1:] {
		client := http.DefaultClient
		values := url.Values{}
		values.Add("file", filename)
		resp, err := client.PostForm(uploadURL, values)
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Printf("%d: %s", resp.StatusCode, resp.Status)
		if resp.StatusCode != 200 {
			fmt.Printf("Status code was not 200: %d", resp.StatusCode)
			Close(resp.Body)
			break
		}
		Close(resp.Body)
	}
}
