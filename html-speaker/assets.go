package htmlspeaker

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"mime"
	"path"
	"path/filepath"
)

//go:embed dist
var assets embed.FS

func ReadAsset(requestPath string) (io.Reader, string, error) {
	f, err := assets.Open(path.Join("dist", requestPath))
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	if stat, _ := f.Stat(); stat.IsDir() {
		return nil, "", errors.New("path is a directory")
	}
	contentType := mime.TypeByExtension(filepath.Ext(requestPath))
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, f); err != nil {
		return nil, "", err
	}
	return &buf, contentType, nil
}

//go:embed testdata
var testDataAssets embed.FS

func OpenTestDataMP3() ([]byte, error) {
	f, err := testDataAssets.Open("testdata/helloworld.mp3")
	if err != nil {
		return nil, fmt.Errorf("failed to open test data: %v", err)
	}
	defer f.Close()
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, f); err != nil {
		return nil, fmt.Errorf("failed to copy test data to buffer: %v", err)
	}
	return buf.Bytes(), nil
}
