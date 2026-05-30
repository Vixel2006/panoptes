package adapter_test

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/Vixel2006/panoptes/internal/adapters"
)

func TestDecompressGzip(t *testing.T) {
	d := adapter.NewDecompressor()
	original := []byte("hello world")
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(original); err != nil {
		t.Fatal(err)
	}
	w.Close()

	result, err := d.Decompress("gzip", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, original) {
		t.Errorf("got %q, want %q", result, original)
	}
}

func TestDecompressDeflateZlib(t *testing.T) {
	d := adapter.NewDecompressor()
	original := []byte("hello deflate")
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(original); err != nil {
		t.Fatal(err)
	}
	w.Close()

	result, err := d.Decompress("deflate", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, original) {
		t.Errorf("got %q, want %q", result, original)
	}
}

func TestDecompressDeflateRaw(t *testing.T) {
	d := adapter.NewDecompressor()
	original := []byte("hello raw deflate")
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	if _, err := w.Write(original); err != nil {
		t.Fatal(err)
	}
	w.Close()

	// raw deflate should fall back after zlib fails
	result, err := d.Decompress("deflate", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, original) {
		t.Errorf("got %q, want %q", result, original)
	}
}

func TestDecompressBrotli(t *testing.T) {
	d := adapter.NewDecompressor()
	original := []byte("hello brotli")
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	if _, err := w.Write(original); err != nil {
		t.Fatal(err)
	}
	w.Close()

	result, err := d.Decompress("br", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, original) {
		t.Errorf("got %q, want %q", result, original)
	}
}

func TestDecompressNoEncodingReturnsBody(t *testing.T) {
	d := adapter.NewDecompressor()
	body := []byte("not compressed at all")
	result, err := d.Decompress("", body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, body) {
		t.Errorf("got %q, want %q", result, body)
	}
}

func TestDecompressUnknownEncodingReturnsBody(t *testing.T) {
	d := adapter.NewDecompressor()
	body := []byte("some random data")
	result, err := d.Decompress("x-my-custom", body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, body) {
		t.Errorf("got %q, want %q", result, body)
	}
}

func TestDecompressNilBody(t *testing.T) {
	d := adapter.NewDecompressor()
	result, err := d.Decompress("gzip", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("got %v, want nil", result)
	}
}

func TestDecompressGarbageEncodingReturnsBody(t *testing.T) {
	d := adapter.NewDecompressor()
	body := []byte("garbage bytes that aren't compressed")
	result, err := d.Decompress("gzip", body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, body) {
		t.Errorf("got %q, want %q", result, body)
	}
}
