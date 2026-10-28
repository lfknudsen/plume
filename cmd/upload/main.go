package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	flag "github.com/spf13/pflag"
)

const defaultURL = "http://localhost:8083/upload"
const configFileName = ".upload.toml"

var https = flag.BoolP("https", "s", false, "Connect via HTTPS.")
var help = flag.BoolP("help", "h", false, "Print this help message and exit.")
var configFile = flag.StringP("config", "c",
	filepath.Join(getExecDir(), configFileName),
	"Set path to the configuration file.")

func printHelp() {
	fmt.Println("Plume client reference implementation\n" +
		"Usage:\n" +
		"\033[96mupload\033[0m [OPTIONS] <filename(s)...>\n" +
		"    Upload files to the server.\n\n" +
		"OPTIONS:")
	flag.PrintDefaults()
}

var config struct {
	URL string `toml:"url"`
}

func getExecDir() string {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exePath := filepath.Dir(exe)
	return exePath
}

func main() {
	flag.Usage = printHelp
	flag.Parse()
	filenames := flag.Args()
	if len(filenames) == 0 || *help {
		printHelp()
		os.Exit(0)
	}

	initialiseConfig()
	err := readConfigFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	if *https {
		withoutPrefix, found := strings.CutPrefix(config.URL, "http://")
		if found {
			config.URL = "https://" + withoutPrefix
		}
	}

	if len(filenames) == 1 {
		uploadOne(filenames[0])
		os.Exit(0)
	}
	uploadMultiple(filenames)
}

func initialiseConfig() {
	config.URL = defaultURL
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

func closeWriter(w *multipart.Writer) {
	if w != nil {
		_ = w.Close()
	}
}

func closeFile(f *os.File) {
	if f != nil {
		_ = f.Close()
	}
}

func Close(body io.ReadCloser) {
	if body != nil {
		_ = body.Close()
	}
}

func Open(filename string) *os.File {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func uploadOne(inputFilename string) {
	var b bytes.Buffer
	mpWriter := multipart.NewWriter(&b)
	file := Open(inputFilename)
	filename := filepath.Base(file.Name())
	fw, err := mpWriter.CreateFormFile("file", filename)
	if err != nil {
		_ = mpWriter.Close()
		closeFile(file)
		log.Fatal(err)
	}

	n, err := io.Copy(fw, file)
	if err != nil {
		_ = mpWriter.Close()
		closeFile(file)
		log.Fatal(err)
	}
	fmt.Printf("Wrote %d bytes from %s\n", n, filename)
	closeFile(file)
	closeWriter(mpWriter)
	req, err := http.NewRequest("POST", config.URL, &b)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", mpWriter.FormDataContentType())
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error occurred:")
		log.Fatal(err)
	}
	fmt.Printf("%d: %s\n", resp.StatusCode, resp.Status)
	if resp.StatusCode != 200 {
		Close(resp.Body)
		log.Fatalf("Status code was not 200, but %d\n", resp.StatusCode)
	}
	Close(resp.Body)
}

func uploadMultiple(filenames []string) {
	config.URL += "_multiple"

	var b bytes.Buffer
	mpWriter := multipart.NewWriter(&b)
	for _, input := range filenames {
		file := Open(input)
		filename := filepath.Base(file.Name())
		fw, err := mpWriter.CreateFormFile("files", filename)
		if err != nil {
			_ = mpWriter.Close()
			closeFile(file)
			log.Fatal(err)
		}

		n, err := io.Copy(fw, file)
		if err != nil {
			_ = mpWriter.Close()
			closeFile(file)
			log.Fatal(err)
		}
		fmt.Printf("Wrote %d bytes from %s\n", n, filename)
		closeFile(file)
	}
	closeWriter(mpWriter)
	req, err := http.NewRequest("POST", config.URL, &b)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", mpWriter.FormDataContentType())
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error occurred:")
		log.Fatal(err)
	}
	fmt.Printf("%d: %s\n", resp.StatusCode, resp.Status)
	if resp.StatusCode != 200 {
		Close(resp.Body)
		log.Fatalf("Status code was not 200, but %d\n", resp.StatusCode)
	}
	Close(resp.Body)
}
