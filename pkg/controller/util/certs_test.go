package util

import (
	"testing"
)

func TestGenerateCACertificate(t *testing.T) {

	c := CertificateSigner{
		RootPath: "/tmp",
	}

	err := c.Initialize()
	if err != nil {
		t.Error(err)
	}

	err = c.GenerateCertificate()
	if err != nil {
		t.Error(err)
	}

}
