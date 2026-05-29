package adapter

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"strings"

	"github.com/andybalholm/brotli"
)

type Decompressor struct{}

func NewDecompressor() *Decompressor {
	return &Decompressor{}
}

func (d *Decompressor) Decompress(encoding string, body []byte) ([]byte, error) {
	encoding = strings.ToLower(encoding)
	if strings.Contains(encoding, "gzip") {
		gr, err := gzip.NewReader(bytes.NewReader(body))
		if err == nil {
			defer gr.Close()
			decompressed, err := io.ReadAll(gr)
			if err == nil {
				return decompressed, nil
			}
		}
	} else if strings.Contains(encoding, "deflate") {
		zr, err := zlib.NewReader(bytes.NewReader(body))
		if err == nil {
			defer zr.Close()
			decompressed, err := io.ReadAll(zr)
			if err == nil {
				return decompressed, nil
			}
		}
		fr := flate.NewReader(bytes.NewReader(body))
		defer fr.Close()
		decompressed, err := io.ReadAll(fr)
		if err == nil {
			return decompressed, nil
		}
	} else if strings.Contains(encoding, "br") {
		br := brotli.NewReader(bytes.NewReader(body))
		decompressed, err := io.ReadAll(br)
		if err == nil {
			return decompressed, nil
		}
	}
	return body, nil
}
