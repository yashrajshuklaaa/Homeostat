// Package v1alpha1 contains the homeostat.dev/v1alpha1 API.
// +kubebuilder:object:generate=true
// +groupName=homeostat.dev
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	GroupVersion   = schema.GroupVersion{Group: "homeostat.dev", Version: "v1alpha1"}
	SchemeBuilder  = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme    = SchemeBuilder.AddToScheme
)
