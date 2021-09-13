package main

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
)

const deck = `AAECAR8GxwPJBLsFmQfZB/gIDI0B2AGoArUDhwSSBe0G6wfbCe0JgQr+DAA=`

func main() {
	ds, err := parseDeckstring(deck)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(ds)
	log.Println(ds.Encode())
}

type deckString struct {
	Version     uint64
	Format      uint64
	HeroDBF     uint64
	SingleCards []uint64
	DoubleCards []uint64
	NCards      []uint64
}

func (ds deckString) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("hero dbf: %d\n", ds.HeroDBF))
	sb.WriteString(fmt.Sprintf("format: %d\n", ds.Format))
	sb.WriteString(fmt.Sprintf("single cards: %v\n", ds.SingleCards))
	sb.WriteString(fmt.Sprintf("double cards: %v\n", ds.DoubleCards))
	sb.WriteString(fmt.Sprintf("n cards: %v", ds.NCards))
	return sb.String()
}

func (ds deckString) Encode() string {
	uvarint := func(v uint64) []byte {
		b := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(b, v)
		return b[:n]
	}

	var buf bytes.Buffer
	buf.Write(uvarint(0)) // magic 0
	buf.Write(uvarint(ds.Version))
	buf.Write(uvarint(ds.Format))
	buf.Write(uvarint(1)) // hero count
	buf.Write(uvarint(ds.HeroDBF))
	buf.Write(uvarint(uint64(len(ds.SingleCards))))
	for i := 0; i < len(ds.SingleCards); i++ {
		buf.Write(uvarint(ds.SingleCards[i]))
	}
	buf.Write(uvarint(uint64(len(ds.DoubleCards))))
	for i := 0; i < len(ds.DoubleCards); i++ {
		buf.Write(uvarint(ds.DoubleCards[i]))
	}
	buf.Write(uvarint(uint64(len(ds.NCards))))
	for i := 0; i < len(ds.NCards); i++ {
		buf.Write(uvarint(ds.NCards[i]))
	}
	return b64.StdEncoding.EncodeToString(buf.Bytes())
}

const validVersion = 1

func parseDeckstring(deck string) (deckString, error) {
	b, err := b64.StdEncoding.DecodeString(deck)
	if err != nil {
		return deckString{}, err
	}

	if b[0] != 0 {
		return deckString{}, fmt.Errorf("invalid header value, expected %c got %C", 0, b[0])
	}

	buf := bytes.NewReader(b[1:])
	var (
		ds          deckString
		singleCount uint64
		doubleCount uint64
		nCount      uint64
	)

	if ds.Version, err = binary.ReadUvarint(buf); err != nil {
		return ds, fmt.Errorf("failed to read version: %v", err)
	}

	if ds.Version != validVersion {
		return ds, fmt.Errorf("invalid version, expected %c got %C", 1, b[1])
	}

	if ds.Format, err = binary.ReadUvarint(buf); err != nil {
		return ds, fmt.Errorf("failed to read format: %v", err)
	}

	// ignore hero count
	if _, err = binary.ReadUvarint(buf); err != nil {
		return ds, fmt.Errorf("failed to read hero count: %v", err)
	}

	if ds.HeroDBF, err = binary.ReadUvarint(buf); err != nil {
		return ds, fmt.Errorf("failed to read hero DBF: %v", err)
	}

	if singleCount, err = binary.ReadUvarint(buf); err != nil {
		return ds, fmt.Errorf("failed to read single quantity card count: %v", err)
	}

	ds.SingleCards = make([]uint64, singleCount)
	for i := uint64(0); i < singleCount; i++ {
		if ds.SingleCards[i], err = binary.ReadUvarint(buf); err != nil {
			return ds, fmt.Errorf("failed to read all single quantity card DBF IDs")
		}
	}

	if doubleCount, err = binary.ReadUvarint(buf); err != nil {
		return ds, fmt.Errorf("failed to read double quantity card count: %v", err)
	}

	ds.DoubleCards = make([]uint64, doubleCount)
	for i := uint64(0); i < doubleCount; i++ {
		if ds.DoubleCards[i], err = binary.ReadUvarint(buf); err != nil {
			return ds, fmt.Errorf("failed to read all double quantity card DBF IDs")
		}
	}

	if nCount, err = binary.ReadUvarint(buf); err != nil {
		return ds, fmt.Errorf("failed to read N quantity card count: %v", err)
	}

	ds.NCards = make([]uint64, nCount)
	for i := uint64(0); i < nCount; i++ {
		if ds.NCards[i], err = binary.ReadUvarint(buf); err != nil {
			return deckString{}, fmt.Errorf("failed to read all N quantity card DBF IDs")
		}
	}

	return ds, nil
}