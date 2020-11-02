package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"
)

var (
	port = flag.Int("port", 8080, "listen addr")
	to = flag.String("to", "data", "directory to save downloads to")
	staticDir = flag.String("static-dir", "static", "directory to serve static assets from")
	mkdir = flag.Bool("mkdir", false, "make directory for [to] if it does not exist (will be equivalent to 'mkdir -p')")
)

func main() {
	flag.Parse()
	if fi, err := os.Stat(*to); err != nil {
		if os.IsNotExist(err)  && *mkdir {
			if err = os.MkdirAll(*to, 0666); err != nil {
				log.Fatalf("failed to create directory: %v", err)
			}
		}
		log.Fatalf("failed to stat (did you forget to use '-mkdir' ?): %v", err)

	} else if !fi.IsDir() {
		log.Fatalf("%q must be a directory", *to)
	}
	log.Printf("saving files to %q\n", *to)

	downloadCounter := NewCounter("download")
	skipCounter := NewCounter("skip")
	mux := http.NewServeMux()
	mux.HandleFunc("/download", download(*to, &downloadCounter, &skipCounter))
	mux.Handle("/static/", noCache(http.StripPrefix("/static/", http.FileServer(http.Dir(*staticDir)))))

	closeCh := make(chan bool)
	go func() {
		log.Println("listening on ", *port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), cors(mux)); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Printf("[ERROR] %v", err)
			}
		}
		closeCh <- true
	}()


	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for termination signal
	select {
	case <-sigChan:
		log.Println("received shutdown request")
		close(closeCh)
	case <-closeCh:
		log.Println("exiting")
	}

	log.Printf("%s\n%s\n", &downloadCounter, &skipCounter)
}

func download(to string, downloadCounter *Counter, skippedCounter *Counter) http.HandlerFunc {
	logResult := func(downloaded uint64, skipped uint64) {
		log.Printf("download: total=%d, complete=%d, skipped=%d\n", downloaded+skipped, downloaded, skipped)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		from := r.FormValue("from")
		if from == "undefined" {
			http.Error(w, "from is 'undefined'", http.StatusBadRequest)
			return
		}
		fileName := path.Base(from)
		outPath := filepath.Join(to, fileName)
		if fileExists(outPath) {
			defer logResult(downloadCounter.Value(), skippedCounter.Increment())
			log.Printf("%q already exists\n", outPath)
			w.WriteHeader(http.StatusNoContent)
			w.Write([]byte("OK"))
			return
		}

		defer func() {
			c := downloadCounter.Increment()
			s := skippedCounter.Value()
			log.Printf("download: total=%d, complete=%d, skipped=%d\n", c+s, c, s)
		}()
		log.Printf("downloading %q from %q\n", fileName, from)

		resp, err := http.Get(from)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()

		// Create the file
		out, err := os.Create(outPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("OK"))
	}
}


func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("Access-Control-Allow-Origin", "*")
		headers.Set("Vary", "Origin")
		if r.Method == "OPTIONS" {
			headers.Add("Vary", "Access-Control-Request-Method")
			headers.Add("Vary", "Access-Control-Request-Headers")
			headers.Set("Access-Control-Allow-Headers", "Content-Type, Origin, Accept")
			headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
			headers.Set("Access-Control-Allow-Credentials", "true")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func noCache(next http.Handler) http.Handler {
	// Taken from https://github.com/mytrile/nocache
	epoch := time.Unix(0, 0).Format(time.RFC1123)

	noCacheHeaders := map[string]string{
		"Expires":         epoch,
		"Cache-Control":   "no-cache, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}

	etagHeaders := []string{
		"ETag",
		"If-Modified-Since",
		"If-Match",
		"If-None-Match",
		"If-Range",
		"If-Unmodified-Since",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delete any ETag headers that may have been set
		for _, v := range etagHeaders {
			if r.Header.Get(v) != "" {
				r.Header.Del(v)
			}
		}
		// Set our NoCache headers
		for k, v := range noCacheHeaders {
			w.Header().Set(k, v)
		}
		next.ServeHTTP(w, r)
	})
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}