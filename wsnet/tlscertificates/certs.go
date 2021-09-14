package tlscertificates

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func LoadCertsFromDirectory(dir string) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("directory traverse error: %w", err)
		}

		if d.IsDir() {
			// Skip directories
			return nil
		}

		foundCerts, err := LoadCertsFromFile(path)
		if err != nil {
			return err
		}
		certs = append(certs, foundCerts...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return certs, nil
}

// LoadCertsFromFile loads all x509 certificates from a given file.
func LoadCertsFromFile(path string) ([]*x509.Certificate, error) {
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
