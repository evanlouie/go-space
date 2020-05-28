package deno

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"runtime"

	"github.com/evanlouie/go-space/pkg/logger"
	"github.com/google/go-github/v31/github"
)

type Context struct {
	DenoPath string
}

// Install Deno locally to a temporary directory
// Modifies DenoPath to point to the Deno executable
func Install() (ctx Context, err error) {
	// Fetch latest Deno release
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), "denoland", "deno")
	if err != nil {
		return ctx, fmt.Errorf("failed to fetch latest Deno GitHub release: %s", err)
	}

	// Determine host OS
	var denoOS string
	var denoBinName string
	switch os := runtime.GOOS; os {
	case "darwin":
		logger.Debug("MacOS detected")
		denoOS = "apple-darwin"
		denoBinName = "deno"
	case "linux":
		logger.Debug("Linux detected")
		denoOS = "unknown-linux-gnu"
		denoBinName = "deno"
	case "windows":
		logger.Debug("Windows detected")
		denoOS = "pc-windows-msvc"
		denoBinName = "deno.exe"
	default:
		return ctx, fmt.Errorf("unsupported OS: %s", os)
	}

	denoUri := fmt.Sprintf("https://github.com/denoland/deno/releases/download/%s/deno-x86_64-%s.zip", *release.TagName, denoOS)

	////////////////////////////////////////////////////////////////////////////////
	// Download latest release to temporary directory
	////////////////////////////////////////////////////////////////////////////////
	// Scaffold temp directory
	denoDir, err := ioutil.TempDir("", "deno")
	if err != nil {
		return ctx, fmt.Errorf("failed to create Deno temporary directory %s: %s", denoDir, err)
	}
	logger.Infof("Downloading Deno %s to %s from %s", *release.TagName, denoDir, denoUri)

	// Download to temp dir
	resp, err := http.Get(denoUri)
	if err != nil {
		return ctx, fmt.Errorf("failed to download Deno from %s: %s", denoUri, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ctx, fmt.Errorf("failed to parse Deno response bytes: %s", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return ctx, fmt.Errorf("failed to create ZipReader: %s", err)
	}

	// Unzip each file to the temporary deno dir
	for _, zipFile := range zipReader.File {
		denoFilepath := path.Join(denoDir, zipFile.Name)
		contents, err := readZipFile(zipFile)
		if err != nil {
			return ctx, fmt.Errorf("failed to read zip file %s", zipFile.Name)
		}
		if err := ioutil.WriteFile(denoFilepath, contents, 0777); err != nil {
			return ctx, fmt.Errorf("failed to write %s to %s", zipFile.Name, denoFilepath)
		}
		logger.Infof("Wrote %s to %s", zipFile.Name, denoFilepath)
	}
	denoBinPath := path.Join(denoDir, denoBinName)
	ctx.DenoPath = denoBinPath

	return ctx, nil
}

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ioutil.ReadAll(f)
}
