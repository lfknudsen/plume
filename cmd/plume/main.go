package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
	flag "github.com/spf13/pflag"
)

const PlumeDirName = "Plume Uploads"

var configFile = ".plume.toml"

var PlumeDir = filepath.Join(getExecDir(), PlumeDirName)

// Command-line arguments
var https = flag.BoolP("https", "s", false, "enable HTTPS mode")
var help = flag.BoolP("help", "h", false, "Print this help message and exit.")
var configPath = flag.StringP("config", "c", filepath.Join(getExecDir(), configFile), "Set path to the configuration file.")

func printHelp() {
	fmt.Println("Plume file upload server\n" +
		"Usage:\n" +
		"\033[96mplume\033[0m [OPTIONS]\n" +
		"    Start the server.\n\n" +
		"OPTIONS:")
	flag.PrintDefaults()
}

// Config contains the key-value pairs found in the configuration file.
var config struct {
	CertDir      string `toml:"cert_dir"`
	CertFilename string `toml:"cert_filename"`
	KeyFilename  string `toml:"key_filename"`
	Host         string `toml:"host"`
	Port         string `toml:"port"`
}

func addr() string {
	return fmt.Sprintf("%s:%s", config.Host, config.Port)
}

func readConfigFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	defer Close(file)
	if err == nil { // If it didn't already exist, we create it with default settings
		log.Printf("No config file found. Creating at %s.\n", filename)
		return toml.NewEncoder(file).Encode(config)
	}
	file, err = os.OpenFile(filename, os.O_RDONLY, 0666)
	defer Close(file)
	if err != nil {
		return err
	}
	return toml.NewDecoder(file).Decode(&config)
}

func initialiseConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		config.CertDir, err = os.Executable()
		if err != nil {
			config.CertDir = "."
		}
	} else {
		config.CertDir = home
	}
	config.Host = "localhost"
	config.Port = "8083"
	config.CertFilename = config.Host + ".pem"
	config.KeyFilename = config.Host + "-key.pem"
}

func main() {
	flag.Usage = printHelp
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	initialiseConfig()
	err := readConfigFile(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(PlumeDir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	engine := gin.Default()
	engine.MaxMultipartMemory = 8 << 20 // 8 MiB
	engine.POST("/upload_multiple", receiveUploadN)
	engine.POST("/upload", receiveUpload)

	if *https {
		certFile := filepath.Join(config.CertDir, config.CertFilename)
		keyFile := filepath.Join(config.CertDir, config.KeyFilename)
		err = engine.RunTLS(addr(), certFile, keyFile)
	} else {
		err = engine.Run(addr())
	}
	if err != nil {
		log.Fatal(err)
	}
}

func receiveUpload(c *gin.Context) {
	log.Println("Received request to upload one file from", c.Request.RemoteAddr)
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
	log.Println("Received request to upload multiple files from", c.Request.RemoteAddr)
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

func Close(f *os.File) {
	if f != nil {
		_ = f.Close()
	}
}
