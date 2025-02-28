package main

import (
	"context"
	"flag"
	"os"

	"github.com/openshift-splat-team/must-hydrate/pkg/controller"
	"github.com/openshift-splat-team/must-hydrate/pkg/server"
	oainstall "github.com/openshift/api"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {

	// Define the flag with a default value
	dataDir := flag.String("data-dir", "/data", "Path to the must-gather directory")
	logDisable := flag.Bool("disable-logs", false, "When true, node resources are not transformed to support log retrieval")

	// Parse command-line arguments
	flag.Parse()

	logf.SetLogger(zap.New())

	log := logf.Log.WithName("main")

	hydrator := &controller.HydratorReconciler{
		RootPath:    *dataDir,
		LogDisabled: *logDisable,
	}
	if err := hydrator.Initialize(context.TODO()); err != nil {
		log.Error(err, "could not initialize hydrator")
		os.Exit(1)
	}

	kubelet := server.KubeletInterfaceServer{
		RootPath: *dataDir,
		Hydrator: hydrator,
	}

	if err := kubelet.Initialize(); err != nil {
		log.Error(err, "could not initialize kubelet server")
		os.Exit(1)
	}
	kubelet.Serve()

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		log.Error(err, "could not create manager")
		os.Exit(1)
	}

	builder.ControllerManagedBy(mgr)
	if err != nil {
		log.Error(err, "could not create controller")
		os.Exit(1)
	}

	hydrator.Client = mgr.GetClient()
	parentScheme := mgr.GetScheme()
	_ = oainstall.Install(parentScheme)
	_ = oainstall.InstallKube(parentScheme)

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "could not start manager")
		os.Exit(1)
	}
}
