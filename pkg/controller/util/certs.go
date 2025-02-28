package util

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path"
	"time"
)

func templateCA() *x509.Certificate {
	return &x509.Certificate{SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Red Hat"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
}

type CertificateSigner struct {
	ca           *x509.Certificate
	caBytes      []byte
	caPrivateKey *rsa.PrivateKey
	RootPath     string
}

func (c *CertificateSigner) Initialize() error {
	err := c.generateCACertificate()
	if err != nil {
		return fmt.Errorf("unable to generate CA certificate")
	}

	return c.PersistCAToPem()
}

func (c *CertificateSigner) generateCACertificate() error {
	ca := templateCA()

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("unable to generate CA private key. %v", err)
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return fmt.Errorf("unable to generate CA private certificate. %v", err)
	}

	c.ca = ca
	c.caBytes = caBytes
	c.caPrivateKey = caPrivKey

	return nil
}

func (c *CertificateSigner) GenerateCertificate(ipAddresses ...net.IP) error {
	cert := templateCA()
	if len(ipAddresses) == 0 {
		ipAddresses = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	}
	cert.IPAddresses = ipAddresses
	cert.NotAfter = time.Now().AddDate(0, 0, 7)

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("unable to generate CA private key. %v", err)
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, cert, c.ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return fmt.Errorf("unable to generate CA private certificate. %v", err)
	}

	return c.PersistToPem(caBytes, caPrivKey)
}

func (c *CertificateSigner) GetPEMs(caBytes []byte, caPrivKey *rsa.PrivateKey) ([]byte, []byte, error) {
	caPEM := new(bytes.Buffer)
	err := pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to convert certificate to PEM: %v", err)
	}

	caPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to convert key to PEM: %v", err)
	}

	return caPEM.Bytes(), caPrivKeyPEM.Bytes(), nil
}

func (c *CertificateSigner) PersistCAToPem() error {
	certPem, _, err := c.GetPEMs(c.caBytes, c.caPrivateKey)
	if err != nil {
		return fmt.Errorf("unable to convert certificate media to PEM. %v", err)
	}
	err = os.WriteFile(path.Join(c.RootPath, "ca.pem"), certPem, 0644)
	if err != nil {
		return fmt.Errorf("unable to write CA certificate PEM. %v", err)
	}

	return nil
}

func (c *CertificateSigner) PersistToPem(certBytes []byte, privKey *rsa.PrivateKey) error {

	certPem, keyPem, err := c.GetPEMs(certBytes, privKey)
	if err != nil {
		return fmt.Errorf("unable to convert certificate media to PEM. %v", err)
	}
	err = os.WriteFile(path.Join(c.RootPath, "cert.pem"), certPem, 0644)
	if err != nil {
		return fmt.Errorf("unable to write certificate PEM. %v", err)
	}
	err = os.WriteFile(path.Join(c.RootPath, "key.pem"), keyPem, 0644)
	if err != nil {
		return fmt.Errorf("unable to write key PEM. %v", err)
	}
	return nil
}
