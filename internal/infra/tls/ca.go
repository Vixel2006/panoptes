package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func LoadOrGenerateCA(configDir string) (*tls.Certificate, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	certPath := filepath.Join(configDir, "panoptes-ca.crt")
	keyPath := filepath.Join(configDir, "panoptes-ca.key")

	if _, errCert := os.Stat(certPath); errCert == nil {
		if _, errKey := os.Stat(keyPath); errKey == nil {
			cert, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err == nil {
				return &cert, nil
			}
		}
	}

	return generateAndServeCA(certPath, keyPath)
}

func generateAndServeCA(certPath, keyPath string) (*tls.Certificate, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"Panoptes Security Engine"},
			CommonName:   "Panoptes Dynamic Root CA",
		},
		NotBefore:            time.Now().Add(-24 * time.Hour),
		NotAfter:             time.Now().AddDate(10, 0, 0),
		IsCA:                 true,
		BasicConstraintsValid: true,
		ExtKeyUsage:          []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:             x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, err
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	defer keyOut.Close()

	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)})

	certOut, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes})

	tlsCert := tls.Certificate{
		Certificate: [][]byte{caBytes},
		PrivateKey:  privKey,
	}

	return &tlsCert, nil
}
