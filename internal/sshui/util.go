package sshui

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: map[string]string{"gr33tz": "tcp.direct"},
		Bytes:   privDER,
	}

	return pem.EncodeToMemory(&privBlock)
}
