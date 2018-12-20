package main

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	extractDir = "graphicPacks"
	rootURL    = "https://github.com/slashiee/cemu_graphic_packs/releases"
)

var (
	gfxPackOnly = []string{
		"BreathOfTheWild",
		"MarioKart8",
		"SuperMario3DWorld",
	}
)

func main() {
	intro()

	url := rootURL + "/latest"
	fmt.Println("loading", url)
	resp, err := http.Get(url)
	if err != nil {
		fail("could not load url:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fail("could not read body:", err)
	}

	path, err := findFilePath(string(body))
	if err != nil {
		fail("could not find path:", err)
	}

	// Check for existing graphic packs
	filename := filepath.Base(path)
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		fmt.Println("your graphic-packs is up-to-date!", filename)
		exit()
	}

	_, err = download(path)
	if err != nil {
		fail("could not download file:", err)
	}

	fmt.Println("extracting", filename, "to", extractDir)
	_, err = unzip(filename, extractDir)
	if err != nil {
		fail("could extraction failed:", err)
	}

	fmt.Println("update done!")
	exit()
}

var re = regexp.MustCompile(`/download/.*/graphicPacks.*.zip`)

func findFilePath(text string) (string, error) {
	matches := re.FindStringSubmatch(text)
	if len(matches) == 0 {
		return "", errors.New("search key no matches")
	}

	return matches[0], nil
}

func download(path string) (filename string, err error) {
	url := rootURL + path
	filename = filepath.Base(url)
	fmt.Println("downloading", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}

	n, err := io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println("downloaded", filename, n/1000, "KB")
	return filename, nil
}

func shouldUnzip(fpath string) bool {
	// Unzip all when not set.
	if len(gfxPackOnly) == 0 {
		return true
	}

	// Search for match and unzip.
	for _, v := range gfxPackOnly {
		ok := strings.Contains(fpath, v)
		if ok {
			return false
		}
	}

	// No match result skip unzip.
	return true
}

func unzip(src string, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Skip selected graphic packs.
		if shouldUnzip(fpath) {
			continue
		}

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			fmt.Println("extracting", fpath)

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()

			if err != nil {
				return filenames, err
			}
		}
	}

	return filenames, nil
}

func intro() {
	fmt.Println("---------------------------------------")
	fmt.Println("\tGraphic Packs Updater")
	fmt.Println("---------------------------------------")
	duration := time.Second
	time.Sleep(duration)
}

func fail(v ...interface{}) {
	fmt.Println(v...)
	exit()
}

func exit() {
	fmt.Print("Press 'Enter' to finish...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	os.Exit(0)
}
