package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/openshift-splat-team/must-hydrate/pkg/controller"
	"github.com/openshift-splat-team/must-hydrate/pkg/controller/util"
)

type KubeletInterfaceServer struct {
	RootPath    string
	Hydrator    *controller.HydratorReconciler
	certManager *util.CertificateSigner
}

func (l *KubeletInterfaceServer) Initialize() error {
	l.certManager = &util.CertificateSigner{
		RootPath: l.RootPath,
	}

	if err := l.certManager.Initialize(); err != nil {
		return fmt.Errorf("unable to initalize the certificate signer. %v", err)
	}

	if err := l.certManager.GenerateCertificate(); err != nil {
		return fmt.Errorf("unable to generate the certificate. %v", err)
	}
	return nil
}

func (l *KubeletInterfaceServer) handle(writer http.ResponseWriter, req *http.Request) {
	logPath, err := l.Hydrator.GetLogPathFromUrl(req.URL)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusNotFound)
		return
	}

	file, err := os.Open(logPath)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	writer.Header().Set("Content-Type", "text/plain")
	writer.Header().Set("Content-Disposition", "attachment; filename=example.txt")
	writer.WriteHeader(http.StatusOK)
	_, err = io.Copy(writer, file)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (l *KubeletInterfaceServer) Serve() {
	go func() {
		http.HandleFunc("/containerLogs/", l.handle)
		err := http.ListenAndServeTLS(":10250", path.Join(l.RootPath, "cert.pem"), path.Join(l.RootPath, "key.pem"), nil)
		if err != nil {
			panic(err)
		}
	}()
}
