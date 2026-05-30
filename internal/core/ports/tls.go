package port

import "crypto/tls"

type CertificateIssuer interface {
	IssueLeaf(hostname string) (*tls.Certificate, error)
}
