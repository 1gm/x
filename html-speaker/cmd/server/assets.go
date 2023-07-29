package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"path/filepath"

	"go.uber.org/zap"
)

//go:embed assets/index.html assets/scripts/*.js assets/testdata/*.mp3
var Assets embed.FS

func handleAsset(log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "Sat, 01 Jan 2000 00:00:00 GMT")

		if r.URL.Path == "/" {
			r.URL.Path = "index.html"
		}

		body, contentType, err := readAsset(r.URL.Path)
		if err == nil {
			w.Header().Set("Content-Type", contentType)
			io.Copy(w, body)
			return
		}
		log.Infof("read asset at %s: %v", r.URL.Path, err)

		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}
}

func readAsset(requestPath string) (io.Reader, string, error) {
	f, err := Assets.Open(path.Join("assets", requestPath))
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

func openTestDataMP3() ([]byte, error) {
	f, err := Assets.Open("testdata/helloworld.mp3")
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
