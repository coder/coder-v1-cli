package certificate

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

func LoadCerts(path string) ([]*x509.Certificate, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var certs []*x509.Certificate
	for i := 0; true; i++ {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}

		// Continue decoding rest in next loop
		data = rest

		if block.Type != "CERTIFICATE" {
			continue // We only want certs
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse cert %d in %q: %w", i, path, err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}
