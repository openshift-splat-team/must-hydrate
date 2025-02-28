package server

import (
	"testing"
	"time"

	"github.com/openshift-splat-team/must-hydrate/pkg/controller/util"
)

func TestHttpServer(t *testing.T) {
	c := util.CertificateSigner{
		RootPath: "/tmp/pem",
	}

	err := c.Initialize()
	if err != nil {
		t.Error(err)
	}

	err = c.GenerateCertificate()
	if err != nil {
		t.Error(err)
	}

	k := KubeletInterfaceServer{
		RootPath: "/tmp/pem",
	}
	k.Initialize()
	k.Serve()

	time.Sleep(20 * time.Second)
}
