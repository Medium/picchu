package webhook

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

type Certificate struct {
	CaCrt []byte
	CaKey []byte
	Crt   []byte
	Key   []byte
}

func (c *Certificate) CABundle() []byte {
	return append(c.CaCrt, c.CaKey...)
}

func GenerateSelfSignedCertificate(hostname string) (*Certificate, error) {
	caPrivKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return nil, err
	}
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Medium"},
			Country:       []string{"US"},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"799 Market Street"},
			PostalCode:    []string{"94103"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	caPrivKeyBytes, err := x509.MarshalECPrivateKey(caPrivKey)
	if err != nil {
		return nil, err
	}
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: caPrivKeyBytes,
	})
	if err != nil {
		return nil, err
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"Medium"},
			Country:       []string{"US"},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"799 Market Street"},
			PostalCode:    []string{"94103"},
		},
		DNSNames:     []string{hostname},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	certPrivKeyBytes, err := x509.MarshalECPrivateKey(certPrivKey)
	if err != nil {
		return nil, err
	}
	err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: certPrivKeyBytes,
	})
	if err != nil {
		return nil, err
	}

	return &Certificate{
		CaCrt: caPEM.Bytes(),
		CaKey: caPrivKeyPEM.Bytes(),
		Crt:   certPEM.Bytes(),
		Key:   certPrivKeyPEM.Bytes(),
	}, nil
}

func MustGenerateSelfSignedCertificate(hostname string) *Certificate {
	c, err := GenerateSelfSignedCertificate(hostname)
	if err != nil {
		panic(err)
	}
	return c
}
