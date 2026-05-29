package port

type Decompressor interface {
	Decompress(encoding string, body []byte) ([]byte, error)
}
