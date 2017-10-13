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
	appsv1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func NewController(ctx *opkit.Context, r opkit.CustomResource) CanaryDeployController {
	return CanaryDeployController{
		context:  ctx,
		resource: r,
	}
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

	logger.Infof("Targetting %s", canaryDeployCopy.Spec.LabelSelectors)

	deployments, err := c.context.Clientset.AppsV1beta1().Deployments("").List(metav1.ListOptions{
		LabelSelector: canaryDeployCopy.Spec.LabelSelectors,
	})
	if err != nil {
		fmt.Printf("Cannot list deployments: %v\n", err)
		return
	}

	desiredImage := canaryDeployCopy.Spec.Image
	//creationTime := canaryDeployCopy.GetCreationTimestamp()

	for _, existing := range deployments.Items {
		// ensure we skip existing canary deploys
		if existing.Labels["role"] == "auto-canary" {
			continue
		}

		logger.Infof("Inspecting deploy/%s", existing.Name)
		actualImage := existing.Spec.Template.Spec.Containers[0].Image
		if actualImage != desiredImage {
			fmt.Printf("Existing: %#v", existing)
			exists, err := c.canaryDeployExists(canaryDeployCopy.Spec.LabelSelectors)
			if err != nil {
				fmt.Printf("Cannot see whether deploy already exists: %v\n", err)
				return
			}
			if !exists {
				c.createCanaryDeployment(existing)
			} else {
				c.updateCanaryDeployment(existing)
			}
			c.scaleDownUserDeployment(existing)
		}
	}
}

func (c *CanaryDeployController) canaryDeployExists(labelSelectors string) (bool, error) {
	deployments, err := c.context.Clientset.AppsV1beta1().Deployments("").List(metav1.ListOptions{
		LabelSelector: labelSelectors + ",role=auto-canary",
	})
	if err != nil {
		fmt.Printf("Cannot list deployments: %v\n", err)
		return false, err
	}
	return len(deployments.Items) > 0, nil
}

func (c *CanaryDeployController) createCanaryDeployment(existing appsv1.Deployment) {
	for {
		replicas := int32(1)

		spec := existing.Spec
		spec.Replicas = &replicas
		spec.Template.Labels["role"] = "auto-canary"

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("canary-%s", existing.ObjectMeta.Name),
				Namespace: existing.Namespace,
			},
			Spec: spec,
		}

		fmt.Printf("%#v\n", deployment)
		if _, err := c.context.Clientset.AppsV1beta1().Deployments(deployment.Namespace).Create(deployment); errors.IsConflict(err) {
			logger.Infof("Encountered conflict, retrying...")
			deployment, err = c.context.Clientset.AppsV1beta1().Deployments("").Get(deployment.Name, metav1.GetOptions{})
			if err != nil {
				logger.Errorf("Fatal error when retrieving deployment: %v", err)
				break
			}
		} else if err != nil {
			logger.Errorf("Fatal error when updating (that's not a conflict): %v", err)
			return
		} else {
			logger.Infof("Successfully created new canary deploy for %s", deployment.Name)
			return
		}
	}
}

func (c *CanaryDeployController) updateCanaryDeployment(deployment appsv1.Deployment) {

}

func (c *CanaryDeployController) scaleDownUserDeployment(deployment appsv1.Deployment) {
	for {
		delta := int32(1)
		newCount := *deployment.Spec.Replicas - delta
		deployment.Spec.Replicas = &newCount
		fmt.Printf("%#v\n", deployment)
		if _, err := c.context.Clientset.AppsV1beta1().Deployments(deployment.Namespace).Update(&deployment); errors.IsConflict(err) {
			logger.Infof("Encountered conflict, retrying...")
			dep, err := c.context.Clientset.AppsV1beta1().Deployments("").Get(deployment.Name, metav1.GetOptions{})
			if err != nil {
				logger.Errorf("Fatal error when retrieving deployment: %v", err)
				break
			}
			deployment = *dep
		} else if err != nil {
			logger.Errorf("Fatal error when updating (that's not a conflict): %v", err)
			return
		} else {
			logger.Infof("Successfully scaled deploy/%s by %d", deployment.Name, delta)
			return
		}
	}
}

func (c *CanaryDeployController) onUpdate(oldObj, newObj interface{}) {
	oldCanaryDeploy := oldObj.(*CanaryDeploy)
	newCanaryDeploy := newObj.(*CanaryDeploy)
	logger.Infof("Updated canaryDeploy '%s' from %s!", newCanaryDeploy.Name, oldCanaryDeploy)
}

func (c *CanaryDeployController) onDelete(obj interface{}) {
	canaryDeploy := obj.(*CanaryDeploy)
	logger.Infof("Deleted canaryDeploy '%s'!", canaryDeploy.Name)
}
