package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	flag "github.com/spf13/pflag"
)

const uploadURL = "localhost:8083"
const PlumeDirName = "Plume Uploads"
const certFileName = "localhost.pem"
const keyFileName = "localhost-key.pem"
const certEnvVar = "PLUME_CERT_DIR"

var certDir = os.Getenv(certEnvVar)
var PlumeDir = filepath.Join(getExecDir(), PlumeDirName)

// Command-line arguments
var https = flag.Bool("https", false, "enable HTTPS mode")

func main() {
	var err error
	err = os.MkdirAll(PlumeDir, 0777)
	if err != nil {
		log.Fatal(err)
	}
	if *https && certDir == "" {
		log.Fatal("PLUME_CERT_DIR environment variable not set")
	}

	engine := gin.Default()
	engine.MaxMultipartMemory = 8 << 20 // 8 MiB
	engine.POST("/upload_multiple", receiveUploadN)
	engine.POST("/upload", receiveUpload)
	if *https {
		certFile := filepath.Join(certDir, certFileName)
		keyFile := filepath.Join(certDir, keyFileName)
		err = engine.RunTLS(uploadURL, certFile, keyFile)
	} else {
		err = engine.Run(uploadURL)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func receiveUpload(c *gin.Context) {
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "parse form err: %s", err.Error())
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "get form err: %s", err.Error())
		return
	}
	filename := sanitiseFilename(file.Filename)
	filename = generateUniqueFilename(PlumeDir, filename)
	out := filepath.Join(PlumeDir, filename)
	err = c.SaveUploadedFile(file, out)
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "save err: %s", err.Error())
		return
	}
	c.String(http.StatusOK, "upload success")
}

// receiveUpload is the handler for POST requests to "/upload"
func receiveUploadN(c *gin.Context) {
	fmt.Println("Received upload request!")
	form, err := c.MultipartForm()
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "get form err: %s", err.Error())
		return
	}
	files := form.File["files"]
	successful := make([]string, len(files))
	for _, file := range files {
		filename := sanitiseFilename(file.Filename)
		filename = generateUniqueFilename(PlumeDir, filename)
		out := filepath.Join(PlumeDir, filename)
		err = c.SaveUploadedFile(file, out)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		successful = append(successful, filename)
	}
	joined := strings.Join(successful, "\n")
	plural := ""
	if len(successful) != 1 {
		plural = "s"
	}
	response := fmt.Sprintf("Successfully uploaded %d file%s:\n%s",
		len(successful), plural, joined)
	c.String(http.StatusOK, response)
}

// As per https://github.com/gin-gonic/gin/issues/1693
func sanitiseFilename(filename string) string {
	filename = filepath.Clean(filename)
	filename = filepath.Base(filename)
	return filename
}

// splitExt returns the stem and extension of the given filename.
func splitExt(filename string) (stem string, ext string) {
	extStart := strings.LastIndex(filename, ".")
	if extStart == -1 {
		return filename, ""
	}
	return filename[:extStart], filename[extStart:]
}

// generateUniqueFilename appends the filename with a (<number>) if a file with
// the given name already exists in PlumeDir.
func generateUniqueFilename(dir string, filename string) string {
	counter := 1
	newFilename := filename
	stem, ext := splitExt(filename)
	_, err := os.Stat(filepath.Join(dir, filename))
	for err == nil {
		newFilename = fmt.Sprintf("%s (%d)%s", stem, counter, ext)
		_, err = os.Stat(filepath.Join(dir, newFilename))
		counter++
	}
	return newFilename
}

func getExecDir() string {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exePath := filepath.Dir(exe)
	return exePath
}
