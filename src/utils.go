package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func DownloadHTTPFile(url string, outFile string) error {
	LogInfo("Downloading %s to %s\n", url, outFile)
	resp, err := http.Get(url)
	if err != nil {
		return FmtError("error downloading file:\n%v", err)
	}
	defer resp.Body.Close()

	fileSize := resp.ContentLength
	LogInfo("File size: %.2f MB\n", float64(fileSize)/(1024*1024))

	out, err := os.Create(outFile)
	if err != nil {
		return FmtError("error creating output file:\n%v", err)
	}
	defer out.Close()

	buf := make([]byte, 10*1024*1024)
	downloaded := int64(0)
	lastProgress := int64(0)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, werr := out.Write(buf[:n])
			if werr != nil {
				return FmtError("error writing to file:\n%v", werr)
			}
			downloaded += int64(n)

			// Show progress every 100MB
			if downloaded-lastProgress >= 100*1024*1024 {
				LogInfo("Downloaded: %.2f MB (%.1f%%)\n",
					float64(downloaded)/(1024*1024),
					float64(downloaded)*100/float64(fileSize))
				lastProgress = downloaded
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return FmtError("error downloading file:\n%v", err)
		}
	}

	LogInfo("Download complete: %.2f MB\n", float64(downloaded)/(1024*1024))
	return nil
}

func SevenZipExtract(archive string, output string) error {
	LogInfo("Extracting %s to %s\n", archive, output)
	cmd := exec.Command("7z", "x", archive, "-o"+output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func AskForConfirmation(msg string) bool {
	LogInfo("\n\033[42m***************************************************************************************************\033[0m")
	fmt.Printf("\033[42m%s [Y/n] (default Y in 5s): \033[0m", msg)
	os.Stdout.Sync()

	ch := make(chan string)
	go func() {
		var input string
		fmt.Scanln(&input)
		if input == "" {
			return
		}
		ch <- input
	}()

	select {
	case response := <-ch:
		os.Stdout.Sync()
		fmt.Println(response)
		return strings.ToLower(response) == "y" || response == ""
	case <-time.After(5 * time.Second):
		os.Stdout.Sync()
		fmt.Println("")
		LogInfo("Timeout - proceeding with default (Y)")
		// Kill the previous goroutine
		go os.Stdin.Write([]byte("y\r\n"))
		go os.Stdin.Sync()
		//close(ch)
		return true
	}
}
