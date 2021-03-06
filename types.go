/*
Copyright 2017 The Rook Authors. All rights reserved.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// schemeGroupVersion is group version used to register these objects
var schemeGroupVersion = schema.GroupVersion{Group: resourceGroup, Version: v1alpha1}

type CanaryDeploy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              CanaryDeploySpec `json:"spec"`
}

type CanaryDeploySpec struct {
	LabelSelectors string        `json:"labelSelectors"`
	Image          string        `json:"image"`
	RolloutPolicy  RolloutPolicy `json:"rolloutPolicy"`
}

type RolloutPolicy struct {
	Duration     string `json:"duration"`
	RateOfChange string `json:"rateOfChange"`
}

type CanaryDeployList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CanaryDeploy `json:"items"`
}
