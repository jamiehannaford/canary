package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reflect"

	"github.com/coreos/pkg/capnslog"
	opkit "github.com/rook/operator-kit"
	"k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var logger = capnslog.NewPackageLogger("github.com/jamiehannaford/canary", "sample")

var (
	schemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	addToScheme   = schemeBuilder.AddToScheme
)

func main() {
	logger.Infof("Getting kubernetes context")
	context, err := createContext()
	if err != nil {
		logger.Errorf("failed to create context. %+v\n", err)
		os.Exit(1)
	}

	// Create and wait for CRD resources
	customResource := opkit.CustomResource{
		Name:    customResourceName,
		Plural:  customResourceNamePlural,
		Group:   resourceGroup,
		Version: v1alpha1,
		Scope:   apiextensionsv1beta1.NamespaceScoped,
		Kind:    reflect.TypeOf(CanaryDeploy{}).Name(),
	}

	logger.Infof("Creating the %s resource", customResourceName)
	resources := []opkit.CustomResource{customResource}
	err = opkit.CreateCustomResources(*context, resources)
	if err != nil {
		logger.Errorf("failed to create custom resource. %+v", err)
		os.Exit(1)
	}

	// create signals to stop watching the resources
	signalChan := make(chan os.Signal, 1)
	stopChan := make(chan struct{})
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// start watching the custom resource
	logger.Infof("Managing the %s resource", customResourceName)
	controller := NewController(context, customResource)
	controller.StartWatch(v1.NamespaceAll, stopChan)

	for {
		select {
		case <-signalChan:
			logger.Infof("shutdown signal received, exiting...")
			close(stopChan)
			return
		}
	}
}

func createContext() (*opkit.Context, error) {
	// create the k8s client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s config. %+v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client. %+v", err)
	}

	apiExtClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s API extension clientset. %+v", err)
	}

	return &opkit.Context{
		Clientset:             clientset,
		APIExtensionClientset: apiExtClientset,
		Interval:              500 * time.Millisecond,
		Timeout:               60 * time.Second,
	}, nil
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(schemeGroupVersion,
		&CanaryDeploy{},
		&CanaryDeployList{},
	)
	metav1.AddToGroupVersion(scheme, schemeGroupVersion)
	return nil
}
