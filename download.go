//go:build ignore

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH

	if len(os.Args) > 2 {
		targetOS = os.Args[1]
		targetArch = os.Args[2]
	}

	fmt.Printf("Fetching localharness for %s/%s...\n", targetOS, targetArch)

	resp, err := http.Get("https://pypi.org/pypi/google-antigravity/json")
	if err != nil {
		panic(fmt.Errorf("failed to fetch PyPI metadata: %v", err))
	}
	defer resp.Body.Close()

	var data struct {
		Urls []struct {
			Filename string `json:"filename"`
			Url      string `json:"url"`
		} `json:"urls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		panic(fmt.Errorf("failed to parse PyPI metadata: %v", err))
	}

	var wheelTag string
	if targetOS == "darwin" && targetArch == "arm64" {
		wheelTag = "macosx_11_0_arm64"
	} else if targetOS == "linux" && targetArch == "amd64" {
		wheelTag = "manylinux_2_17_x86_64"
	} else if targetOS == "linux" && targetArch == "arm64" {
		wheelTag = "manylinux_2_17_aarch64"
	} else {
		panic(fmt.Errorf("unsupported OS/Arch combination: %s/%s", targetOS, targetArch))
	}

	var wheelUrl string
	for _, u := range data.Urls {
		if strings.HasSuffix(u.Filename, ".whl") && strings.Contains(u.Filename, wheelTag) {
			wheelUrl = u.Url
			break
		}
	}

	if wheelUrl == "" {
		panic(fmt.Errorf("no wheel found for tag: %s", wheelTag))
	}

	fmt.Printf("Downloading wheel from: %s\n", wheelUrl)

	wheelResp, err := http.Get(wheelUrl)
	if err != nil {
		panic(fmt.Errorf("failed to download wheel: %v", err))
	}
	defer wheelResp.Body.Close()

	wheelData, err := io.ReadAll(wheelResp.Body)
	if err != nil {
		panic(fmt.Errorf("failed to read wheel data: %v", err))
	}

	zipReader, err := zip.NewReader(bytes.NewReader(wheelData), int64(len(wheelData)))
	if err != nil {
		panic(fmt.Errorf("failed to open zip: %v", err))
	}

	var harnessFile *zip.File
	for _, f := range zipReader.File {
		if strings.HasSuffix(f.Name, "google/antigravity/bin/localharness") {
			harnessFile = f
			break
		}
	}

	if harnessFile == nil {
		panic("localharness binary not found inside wheel")
	}

	inFile, err := harnessFile.Open()
	if err != nil {
		panic(fmt.Errorf("failed to open file inside zip: %v", err))
	}
	defer inFile.Close()

	os.MkdirAll("bin", 0755)
	outPath := filepath.Join("bin", "localharness")
	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		panic(fmt.Errorf("failed to create output file: %v", err))
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, inFile); err != nil {
		panic(fmt.Errorf("failed to write localharness: %v", err))
	}

	fmt.Printf("Successfully extracted localharness to %s\n", outPath)
}
