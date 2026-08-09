package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RecommendationPhase string

const (
	PhasePending  RecommendationPhase = "Pending"
	PhaseAdmitted RecommendationPhase = "Admitted"
	PhaseBlocked  RecommendationPhase = "Blocked"
	PhaseApplied  RecommendationPhase = "Applied"
	PhaseFailed   RecommendationPhase = "Failed"
)

// TargetRef points at the workload a recommendation is about.
type TargetRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
}

// ResourceDelta is a proposed change to one container's requests/limits.
type ResourceDelta struct {
	Container        string              `json:"container"`
	CurrentRequests  corev1.ResourceList `json:"currentRequests,omitempty"`
	ProposedRequests corev1.ResourceList `json:"proposedRequests"`
	// DeltaPercent = max(|proposed-current|/current) across resource types.
	// Kyverno checks this field directly, so keep it in sync with whatever
	// computes the actual diff.
	DeltaPercent int32 `json:"deltaPercent"`
}

type HomeostatRecommendationSpec struct {
	Target    TargetRef `json:"target"`
	AgentName string    `json:"agentName"`
	Reason    string    `json:"reason"`

	ResourceDeltas []ResourceDelta `json:"resourceDeltas"`

	// RequiresLabels gates admission on namespace labels being present,
	// e.g. finance approval before touching GPU node pools.
	RequiresLabels map[string]string `json:"requiresLabels,omitempty"`
}

type HomeostatRecommendationStatus struct {
	Phase              RecommendationPhase `json:"phase,omitempty"`
	Message            string              `json:"message,omitempty"`
	ObservedGeneration int64               `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition  `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.target.name`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// HomeostatRecommendation is a proposed optimization from a kagent agent.
// It's never applied directly - it gets staged here, Kyverno validates it,
// and only an Admitted recommendation gets turned into an actual HPA/VPA
// patch. That staging step is the whole point: every autonomous change has
// both an agent decision and a policy decision attached to it.
type HomeostatRecommendation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HomeostatRecommendationSpec   `json:"spec"`
	Status HomeostatRecommendationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type HomeostatRecommendationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HomeostatRecommendation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HomeostatRecommendation{}, &HomeostatRecommendationList{})
}
