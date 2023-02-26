package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/1gm/x/internal/log"
)

func main() {
	out := flag.String("o", "data", "directory to save results")
	compress := flag.Bool("c", false, "gzip data before saving")
	mkdir := flag.Bool("mkdir", false, "make directory for '-o' if it does not exist (will be equivalent to 'mkdir -p')")
	flag.Parse()

	exitCode := realMain(*out, *mkdir, *compress)
	os.Exit(exitCode)
}
func realMain(directory string, createDirectory bool, compress bool) int {
	log := log.New()
	defer log.Sync()

	if fi, err := os.Stat(directory); err != nil {
		if os.IsNotExist(err) && createDirectory {
			if err = os.MkdirAll(directory, 0666); err != nil {
				log.Errorf("failed to create directory: %v", err)
				return 1
			}
		}
		log.Errorf("failed to stat (did you forget to use '-mkdir' ?): %v", err)
		return 1
	} else if !fi.IsDir() {
		log.Errorf("%q must be a directory", directory)
		return 1
	}
	log.Infof("results will be written into %q", directory)

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() { <-c; cancel() }()

	for {
		now := time.Now().UTC().Format(time.RFC3339)
		log.Info("fetching rates at ", now)
		pairs, raw, err := fetchTickerPairs(ctx)
		if err != nil {
			log.Error(err)
			return 1
		}

		var buf *bytes.Buffer
		filename := filepath.Join(directory, now+".json")
		if compress {
			filename += ".gz"
			if err = gzipCompress(buf, raw); err != nil {
				log.Error("failed to compress raw response: ", err)
				return 1
			}
		} else {
			buf = bytes.NewBuffer(raw)
		}

		log.Info("saving rates to ", filename)
		if err = os.WriteFile(filename, buf.Bytes(), os.ModePerm); err != nil {
			log.Error("failed to save response: ", err)
			return 1
		}

		log.Info("BTC/USD rate (today): ", pairs["XXBTZUSD"].VolumeWeightedAveragePrice[0])
		log.Info("BTC/USD rate (24 hours): ", pairs["XXBTZUSD"].VolumeWeightedAveragePrice[1])

		select {
		case <-time.After(time.Minute):

		case <-ctx.Done():
			log.Info("shutting down")
			return 0
		}
	}

	return 0
}

func fetchTickerPairs(ctx context.Context) (pairs map[string]tickerPair, raw []byte, err error) {
	apiRes, err := http.DefaultClient.Get("https://api.kraken.com/0/public/Ticker")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get ticker: %v", err)
	}

	b, err := io.ReadAll(apiRes.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %v", err)
	} else if err = apiRes.Body.Close(); err != nil {
		return nil, nil, fmt.Errorf("failed to close response body: %v", err)
	}

	var res tickerResponse
	if err = json.NewDecoder(bytes.NewReader(b)).Decode(&res); err != nil {
		return nil, nil, fmt.Errorf("failed to decode ticker response body: %v", err)
	}

	if len(res.Error) > 0 {
		return nil, nil, fmt.Errorf("ticker api returned errors: %s", strings.Join(res.Error, "\n"))
	}

	return res.Result, b, nil
}

type tickerResponse struct {
	Error  []string              `json:"error"`
	Result map[string]tickerPair `json:"result"`
}

type tickerPair struct {
	// Ask array(<price>, <whole lot volume>, <lot volume>)
	Ask []string `json:"a"`
	// Bid array(<price>, <whole lot volume>, <lot volume>)
	Bid []string `json:"b"`
	// Last trade closed array(<price>, <lot volume>)
	Close []string `json:"c"`
	// Volume array(<today>, <last 24 hours>)
	Volume []string `json:"v"`
	// Volume weighted average price array(<today>, <last 24 hours>)
	VolumeWeightedAveragePrice []string `json:"p"`
	// Number of trades array(<today>, <last 24 hours>)
	Trades []int `json:"t"`
	// Low array(<today>, <last 24 hours>)
	Low []string `json:"l"`
	// High array(<today>, <last 24 hours>)
	High []string `json:"h"`
	// Today's opening price
	OpeningPrice float64 `json:"o,string"`
}

func gzipCompress(w io.Writer, b []byte) error {
	gz := gzip.NewWriter(w)
	if _, err := gz.Write(b); err != nil {
		return err
	}
	return gz.Close()
}
