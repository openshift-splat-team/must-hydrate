package util

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func GetGvkKey(a schema.GroupVersionKind) string {
	return fmt.Sprintf("%s.%s.%s", a.Group, a.Version, a.Kind)
}

func IsGvk(a schema.GroupVersionKind, b schema.GroupVersionKind) bool {
	return a.Kind == b.Kind &&
		a.Group == b.Group &&
		a.Version == b.Version
}

func WriteKubeconfig(cfg *rest.Config) error {
	clusterName := "envtest"
	contextName := fmt.Sprintf("%s@%s", cfg.Username, clusterName)
	c := api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   cfg.Host,
				CertificateAuthorityData: cfg.CAData,
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: cfg.Username,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			cfg.Username: {
				ClientKeyData:         cfg.KeyData,
				ClientCertificateData: cfg.CertData,
			},
		},
		CurrentContext: contextName,
	}
	data, err := clientcmd.Write(c)
	if err != nil {
		return fmt.Errorf("unable to write kubeconfig. %v", err)
	}

	err = os.WriteFile("envtest.kubeconfig", data, 0666)
	if err != nil {
		return fmt.Errorf("unable to write kubeconfig to disk. %v", err)
	}
	if _, err := os.Stat("/data"); err == nil {
		_ = os.WriteFile("/data/envtest.kubeconfig", data, 0666)
	}
	return nil
}
