package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	homeostatv1alpha1 "github.com/yashrajshuklaaa/homeostat/api/v1alpha1"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding appsv1 to scheme: %v", err)
	}
	if err := homeostatv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding homeostat v1alpha1 to scheme: %v", err)
	}
	return scheme
}

func TestReconcile_AdmittedPatchesDeployment(t *testing.T) {
	scheme := newScheme(t)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Resources: corev1.ResourceRequirements{}},
					},
				},
			},
		},
	}

	rec := &homeostatv1alpha1.HomeostatRecommendation{
		ObjectMeta: metav1.ObjectMeta{Name: "rec-1", Namespace: "default"},
		Spec: homeostatv1alpha1.HomeostatRecommendationSpec{
			Target: homeostatv1alpha1.TargetRef{
				APIVersion: "apps/v1", Kind: "Deployment", Name: "checkout", Namespace: "default",
			},
			AgentName: "optimization-agent",
			Reason:    "p95 memory usage 40% below request",
			ResourceDeltas: []homeostatv1alpha1.ResourceDelta{
				{
					Container:        "app",
					ProposedRequests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")},
					DeltaPercent:     20,
				},
			},
		},
		Status: homeostatv1alpha1.HomeostatRecommendationStatus{Phase: homeostatv1alpha1.PhaseAdmitted},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(dep, rec).
		WithStatusSubresource(&homeostatv1alpha1.HomeostatRecommendation{}).
		Build()

	r := &HomeostatRecommendationReconciler{Client: c}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "rec-1", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	var got homeostatv1alpha1.HomeostatRecommendation
	if err := c.Get(context.Background(), types.NamespacedName{Name: "rec-1", Namespace: "default"}, &got); err != nil {
		t.Fatalf("fetching recommendation after reconcile: %v", err)
	}
	if got.Status.Phase != homeostatv1alpha1.PhaseApplied {
		t.Errorf("expected phase Applied, got %s (message: %s)", got.Status.Phase, got.Status.Message)
	}

	var gotDep appsv1.Deployment
	if err := c.Get(context.Background(), types.NamespacedName{Name: "checkout", Namespace: "default"}, &gotDep); err != nil {
		t.Fatalf("fetching deployment after reconcile: %v", err)
	}
	want := resource.MustParse("256Mi")
	got2 := gotDep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory]
	if got2.Cmp(want) != 0 {
		t.Errorf("expected memory request %s, got %s", want.String(), got2.String())
	}
}

func TestReconcile_BlockedDoesNothing(t *testing.T) {
	scheme := newScheme(t)

	rec := &homeostatv1alpha1.HomeostatRecommendation{
		ObjectMeta: metav1.ObjectMeta{Name: "rec-2", Namespace: "default"},
		Spec: homeostatv1alpha1.HomeostatRecommendationSpec{
			Target:    homeostatv1alpha1.TargetRef{Kind: "Deployment", Name: "checkout", Namespace: "default"},
			AgentName: "optimization-agent",
		},
		Status: homeostatv1alpha1.HomeostatRecommendationStatus{
			Phase:   homeostatv1alpha1.PhaseBlocked,
			Message: "delta exceeds 30% threshold",
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(rec).Build()
	r := &HomeostatRecommendationReconciler{Client: c}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "rec-2", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	var got homeostatv1alpha1.HomeostatRecommendation
	if err := c.Get(context.Background(), types.NamespacedName{Name: "rec-2", Namespace: "default"}, &got); err != nil {
		t.Fatalf("fetching recommendation after reconcile: %v", err)
	}
	if got.Status.Phase != homeostatv1alpha1.PhaseBlocked {
		t.Errorf("expected phase to remain Blocked, got %s", got.Status.Phase)
	}
}

// TestReconcile_PendingTransitionsToAdmitted covers the core behavior
// change from ADR 0001: since Kyverno's Enforce policy validates
// synchronously at creation time, an object that exists has already
// passed policy. A freshly-created recommendation with no phase set
// should be marked Admitted on its first reconcile, not left untouched.
func TestReconcile_PendingTransitionsToAdmitted(t *testing.T) {
	scheme := newScheme(t)

	rec := &homeostatv1alpha1.HomeostatRecommendation{
		ObjectMeta: metav1.ObjectMeta{Name: "rec-3", Namespace: "default"},
		Spec: homeostatv1alpha1.HomeostatRecommendationSpec{
			Target:    homeostatv1alpha1.TargetRef{Kind: "Deployment", Name: "checkout", Namespace: "default"},
			AgentName: "optimization-agent",
		},
		// Phase left unset - simulates a freshly-created recommendation
		// that has just passed Kyverno's admission check.
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rec).
		WithStatusSubresource(&homeostatv1alpha1.HomeostatRecommendation{}).
		Build()
	r := &HomeostatRecommendationReconciler{Client: c}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "rec-3", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if !result.Requeue {
		t.Errorf("expected Requeue: true so the Admitted branch runs next, got %+v", result)
	}

	var got homeostatv1alpha1.HomeostatRecommendation
	if err := c.Get(context.Background(), types.NamespacedName{Name: "rec-3", Namespace: "default"}, &got); err != nil {
		t.Fatalf("fetching recommendation after reconcile: %v", err)
	}
	if got.Status.Phase != homeostatv1alpha1.PhaseAdmitted {
		t.Errorf("expected phase Admitted, got %s", got.Status.Phase)
	}
}
