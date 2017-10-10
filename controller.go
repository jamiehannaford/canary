/*
Copyright 2016 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Some of the code below came from https://github.com/coreos/etcd-operator
which also has the apache 2.0 license.
*/

// Package main for a sample operator
package main

import (
	"fmt"

	opkit "github.com/rook/operator-kit"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

const (
	customResourceName       = "canarydeploy"
	customResourceNamePlural = "canarydeploys"
	resourceGroup            = "mycompany.io"
	v1alpha1                 = "v1alpha1"
)

// CanaryDeployController represents a controller object for sample custom resources
type CanaryDeployController struct {
	context  *opkit.Context
	scheme   *runtime.Scheme
	resource opkit.CustomResource
}

// Watch watches for instances of CanaryDeploy custom resources and acts on them
func (c *CanaryDeployController) StartWatch(namespace string, stopCh chan struct{}) error {
	client, scheme, err := opkit.NewHTTPClient(resourceGroup, v1alpha1, schemeBuilder)
	if err != nil {
		return fmt.Errorf("failed to get a k8s client for watching sample resources: %v", err)
	}
	c.scheme = scheme

	resourceHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}
	watcher := opkit.NewWatcher(c.resource, namespace, resourceHandlers, client)
	go watcher.Watch(&CanaryDeploy{}, stopCh)
	return nil
}

func (c *CanaryDeployController) onAdd(obj interface{}) {
	canaryDeploy := obj.(*CanaryDeploy)

	// Never modify objects from the store. It's a read-only, local cache.
	// Use scheme.Copy() to make a deep copy of original object.
	copyObj, err := c.scheme.Copy(canaryDeploy)
	if err != nil {
		fmt.Printf("failed to create a deep copy of canaryDeploy object: %v\n", err)
		return
	}
	canaryDeployCopy := copyObj.(*CanaryDeploy)

	logger.Infof("Added canaryDeploy '%s' with Hello=%s!", canaryDeployCopy.Name, canaryDeployCopy.Spec.Hello)
}

func (c *CanaryDeployController) onUpdate(oldObj, newObj interface{}) {
	oldCanaryDeploy := oldObj.(*CanaryDeploy)
	newCanaryDeploy := newObj.(*CanaryDeploy)
	logger.Infof("Updated canaryDeploy '%s' from %s to %s!", newCanaryDeploy.Name, oldCanaryDeploy.Spec.Hello, newCanaryDeploy.Spec.Hello)
}

func (c *CanaryDeployController) onDelete(obj interface{}) {
	canaryDeploy := obj.(*CanaryDeploy)
	logger.Infof("Deleted canaryDeploy '%s' with Hello=%s!", canaryDeploy.Name, canaryDeploy.Spec.Hello)
}
