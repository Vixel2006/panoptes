package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"time"
)

type CertificateGenerator struct {
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
}

func NewCertificateGenerator(tlsCert *tls.Certificate) (*CertificateGenerator, error) {
	x509Cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, err
	}

	rsaKey, ok := tlsCert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not valid RSA")
	}

	return &CertificateGenerator{
		caCert: x509Cert,
		caKey:  rsaKey,
	}, nil
}

func (cg *CertificateGenerator) IssueLeaf(hostname string) (*tls.Certificate, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"Panoptes Security Engine"},
			CommonName:   hostname,
		},
		DNSNames:    []string{hostname},
		NotBefore:   time.Now().Add(-1 * time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, cg.caCert, &privKey.PublicKey, cg.caKey)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: [][]byte{certBytes, cg.caCert.Raw},
		PrivateKey:  privKey,
	}, nil
}
