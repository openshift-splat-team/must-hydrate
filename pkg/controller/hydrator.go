package controller

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/openshift-splat-team/must-hydrate/pkg/controller/util"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	metadataKeysToDrop = []string{"creationTimestamp", "managedFields", "uid", "resourceVersion", "generation"}
	kindPriority       = []schema.GroupVersionKind{
		{
			Group:   "apiextensions.k8s.io",
			Version: "v1",
			Kind:    "CustomResourceDefinition",
		},
		{
			Group:   "",
			Version: "v1",
			Kind:    "Namespace",
		},
		{
			Group:   "",
			Version: "v1",
			Kind:    "Node",
		},
		{
			Group:   "config.openshift.io",
			Version: "v1",
			Kind:    "ClusterOperator",
		},
		{
			Group:   "config.openshift.io",
			Version: "v1",
			Kind:    "ClusterVersion",
		},
	}
	dontHydrateKinds = []schema.GroupVersionKind{
		{
			Group:   "admissionregistration.k8s.io",
			Version: "v1",
			Kind:    "ValidatingWebhookConfiguration",
		},
		{
			Group:   "",
			Version: "v1",
			Kind:    "Secret",
		},
		{
			Group:   "",
			Version: "v1",
			Kind:    "Service",
		},
		{
			Group:   "build.openshift.io",
			Version: "v1",
			Kind:    "BuildConfig",
		},
		{
			Group:   "cns.vmware.com",
			Version: "v1alpha1",
			Kind:    "CnsVolumeOperationRequest",
		},
		{
			Group:   "cns.vmware.com",
			Version: "v1alpha1",
			Kind:    "CSINodeTopology",
		},
		{
			Group:   "oauth.openshift.io",
			Version: "v1",
			Kind:    "OAuthClient",
		},
		{
			Group:   "operators.coreos.com",
			Version: "v1",
			Kind:    "OperatorGroup",
		},
		{
			Group:   "route.openshift.io",
			Version: "v1",
			Kind:    "RouteList",
		},
		{
			Group:   "route.openshift.io",
			Version: "v1",
			Kind:    "Route",
		},
		{
			Group:   "user.openshift.io",
			Version: "v1",
			Kind:    "User",
		},
		{
			Group:   "metrics.k8s.io",
			Version: "v1beta1",
			Kind:    "Metrics",
		},
		{
			Group:   "template.openshift.io",
			Version: "v1",
			Kind:    "Template",
		},
	}
)

type GvkCacheItem struct {
	schema.GroupVersionKind

	instances []*unstructured.Unstructured
}

// HydratorReconciler is a simple ControllerManagedBy example implementation.
type HydratorReconciler struct {
	client.Client

	RootPath      string
	log           logr.Logger
	testEnv       *envtest.Environment
	dynamicClient *dynamic.DynamicClient
	clientSet     *kubernetes.Clientset
	context       context.Context
	restConfig    *rest.Config

	gvkCache map[string]*GvkCacheItem
}

func (a *HydratorReconciler) loadResources() error {
	var yamlFiles []string
	rootDir := a.RootPath
	a.gvkCache = make(map[string]*GvkCacheItem)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
			err = a.prepareAndCacheResource(path)
			if err != nil {
				return err
			}
			yamlFiles = append(yamlFiles, path)
		}
		return nil
	})
	if err != nil {
		a.log.Error(err, "error walking the path", "rootDir", rootDir)
		return fmt.Errorf("error walking the path. %v", err)
	}

	return nil
}

func (a *HydratorReconciler) cleanupMetadata(root map[string]any) {
	if metadata, ok := root["metadata"].(map[string]any); ok {
		if len(metadata) != 0 {
			for _, keyToDrop := range metadataKeysToDrop {
				delete(metadata, keyToDrop)
			}
			root["metadata"] = metadata
		}
	}

	for _, v := range root {
		if _, ok := v.(map[string]any); ok {
			a.cleanupMetadata(v.(map[string]any))
		}
	}
}

func (a *HydratorReconciler) shouldNotHydrate(resource unstructured.Unstructured) bool {
	for _, skipResource := range dontHydrateKinds {
		if util.IsGvk(skipResource, resource.GroupVersionKind()) {
			return true
		}
	}

	return false
}

func (a *HydratorReconciler) cacheResources(resources []unstructured.Unstructured) {
	for _, resource := range resources {
		if a.shouldNotHydrate(resource) {
			gvk := resource.GroupVersionKind()
			a.log.V(4).Info("skipping hydrating resource with type", "group", gvk.Group, "version", gvk.Version, "kind", gvk.Kind)
			continue
		}

		var cachedResource *GvkCacheItem
		var exists bool

		key := util.GetGvkKey(resource.GroupVersionKind())

		gvk := resource.GroupVersionKind()

		if cachedResource, exists = a.gvkCache[key]; !exists {
			cachedResource = &GvkCacheItem{
				GroupVersionKind: gvk,
				instances:        []*unstructured.Unstructured{},
			}
		}

		cachedResource.instances = append(cachedResource.instances, &resource)
		a.gvkCache[key] = cachedResource
	}
}

func (a *HydratorReconciler) prepareAndCacheResource(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	// Unmarshal YAML into a map.
	var content map[string]interface{}
	if err := yaml.Unmarshal(data, &content); err != nil {
		return fmt.Errorf("error unmarshalling YAML: %v", err)
	}

	a.cleanupMetadata(content)

	updatedYaml, err := yaml.Marshal(content)
	if err != nil {
		return fmt.Errorf("error marshalling YAML: %v", err)
	}

	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(updatedYaml), 4096)
	var obj unstructured.Unstructured
	if err := decoder.Decode(&obj); err != nil {
		a.log.Error(err, "error decoding YAML")
	}

	var resources []unstructured.Unstructured

	if obj.IsList() {
		items, found, err := unstructured.NestedSlice(obj.Object, "items")
		if err != nil || !found {
			a.log.Error(err, "error retrieving items from list")
		}

		for _, item := range items {
			item, ok := item.(map[string]any)

			if !ok {
				a.log.Info("skipping an item due to type assertion failure")
			}

			resources = append(resources, unstructured.Unstructured{Object: item})

		}

	} else {
		resources = append(resources, obj)
	}

	a.cacheResources(resources)

	return nil
}

func (a *HydratorReconciler) applyResources(applyGvks ...schema.GroupVersionKind) error {
	var unapplied []*unstructured.Unstructured

	for key, gvkCacheItem := range a.gvkCache {
		if len(applyGvks) > 0 {
			var apply bool
			for _, applyGvk := range applyGvks {
				if util.IsGvk(applyGvk, gvkCacheItem.GroupVersionKind) {
					apply = true
					break
				}
			}
			if !apply {
				continue
			}
		}
		a.log.Info("applying gvk", "gvk", util.GetGvkKey(gvkCacheItem.GroupVersionKind), "remaining", len(gvkCacheItem.instances))
		for _, resourceInstance := range gvkCacheItem.instances {
			gvk := resourceInstance.GroupVersionKind()
			resourceIface, err := util.New(a.restConfig, gvk, resourceInstance.GetNamespace())
			if err != nil {
				a.log.Error(err, "unable to create resource interface", "gvk", util.GetGvkKey(gvk), "name", resourceInstance.GetName())
				break
			}

			existing, err := resourceIface.Get(a.context, resourceInstance.GetName(), metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				klog.V(2).Infof("%s %s/%s not found, creating", resourceInstance.GetKind(), resourceInstance.GetNamespace(), resourceInstance.GetName())
				a.cleanupMetadata(resourceInstance.Object)
				existing, err = resourceIface.Create(a.context, resourceInstance, v1.CreateOptions{})
				if err != nil {
					a.log.Error(err, "unable to create resource", "gvk", util.GetGvkKey(gvk), "name", resourceInstance.GetName())
					unapplied = append(unapplied, resourceInstance)
					continue
				}
			}

			if status, ok := resourceInstance.Object["status"]; ok {
				existing.Object["status"] = status
				_, err = resourceIface.UpdateStatus(a.context, existing, v1.UpdateOptions{})
				if err != nil {
					a.log.Error(err, "unable to udpate status for resource", "gvk", util.GetGvkKey(gvk), "name", resourceInstance.GetName())
					unapplied = append(unapplied, existing)
					continue
				}
			}
		}
		gvkCacheItem.instances = unapplied
		a.gvkCache[key] = gvkCacheItem
	}

	return nil
}

func (a *HydratorReconciler) Initialize(ctx context.Context) error {
	var err error

	a.context = ctx

	logf.SetLogger(zap.New())

	a.log = logf.Log.WithName("HydratorReconciler")

	if len(a.RootPath) == 0 {
		a.RootPath = "./data"
	}

	err = a.loadResources()

	if err != nil {
		err = fmt.Errorf("unable to load resources %v", err)
		a.log.Error(err, err.Error())
		return err
	}
	api := envtest.APIServer{}
	api.Configure().Set("service-cluster-ip-range", "192.0.0.0/24")
	a.testEnv = &envtest.Environment{
		// Provide paths to your CRD definitions if needed.
		CRDDirectoryPaths:        []string{},
		AttachControlPlaneOutput: true,
		ControlPlane: envtest.ControlPlane{
			APIServer: &api,
		},
	}

	cfg, err := a.testEnv.Start()
	if err != nil {
		a.log.Error(err, "unable to start envTest")
		return fmt.Errorf("unable to start envTest: %v", err)
	}
	a.restConfig = cfg
	a.dynamicClient, err = dynamic.NewForConfig(cfg)
	if err != nil {
		a.log.Error(err, "error creating dynamic client")
	}

	a.clientSet, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create the k8s client set. %v", err)
	}

	err = util.WriteKubeconfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to write kubeconfig: %v", err)
	}
	go a.Reconcile()

	return nil
}

func (a *HydratorReconciler) Reconcile() {
	var priorityDone bool
	var err error
	backoff := 1

	for {
		if !priorityDone {
			err = a.applyResources(kindPriority...)
			if err != nil {
				a.log.Error(err, "unable to apply all priority resources")
			} else {
				a.log.Info("applied all priority resources")
				priorityDone = true
			}
		}

		err = a.applyResources()
		if err != nil {
			a.log.Error(err, "unable to apply all resources")
		} else {
			a.log.Info("applied all resources. exiting reconciliation")
			return
		}
		seconds := 1 << backoff
		a.log.Info("backing off", "seconds", seconds)
		time.Sleep(time.Duration(seconds) * time.Second)
		backoff++
	}
}
