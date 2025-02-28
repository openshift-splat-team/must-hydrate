package util

import (
	"fmt"
	"os"
	"path"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// GetGvkKey returns a string composed of the Group, Version, and Kind of a schema.GroupVersionKind.
func GetGvkKey(a schema.GroupVersionKind) string {
	return fmt.Sprintf("%s.%s.%s", a.Group, a.Version, a.Kind)
}

// IsGvk checks if two GroupVersionKinds are equal.
func IsGvk(a schema.GroupVersionKind, b schema.GroupVersionKind) bool {
	return a.Kind == b.Kind &&
		a.Group == b.Group &&
		a.Version == b.Version
}

// WriteKubeconfig writes a kubeconfig file to the specified path.
//
// Parameters:
// - cfg: The configuration to write to the kubeconfig file.
// - outPath: The path to write the kubeconfig file to.
//
// Returns:
// - error: An error if the kubeconfig file could not be written.
func WriteKubeconfig(cfg *rest.Config, outPath string) error {
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

	err = os.WriteFile(path.Join(outPath, "envtest.kubeconfig"), data, 0666)
	if err != nil {
		return fmt.Errorf("unable to write kubeconfig to disk. %v", err)
	}
	if _, err := os.Stat("/data"); err == nil {
		_ = os.WriteFile("/data/envtest.kubeconfig", data, 0666)
	}
	return nil
}
