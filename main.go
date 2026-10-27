package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func getExecDir() string {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exePath := filepath.Dir(exe)
	return exePath
}

const PlumeDirName = "Plume Uploads"

var PlumeDir = filepath.Join(getExecDir(), PlumeDirName)

func main() {
	var err error
	err = os.MkdirAll(PlumeDir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	engine := gin.Default()
	engine.POST("/upload", receiveUpload)
	err = engine.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// receiveUpload is the handler for POST requests to "/upload"
func receiveUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = c.SaveUploadedFile(file, file.Filename)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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
func generateUniqueFilename(filename string) string {
	counter := 1
	newFilename := filename
	stem, ext := splitExt(filename)
	_, err := os.Stat(filepath.Join(PlumeDir, filename))
	for err == nil {
		newFilename = fmt.Sprintf("%s (%d)%s", stem, counter, ext)
		_, err = os.Stat(filepath.Join(PlumeDir, newFilename))
	}
	return newFilename
}
