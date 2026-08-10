package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	homeostatv1alpha1 "github.com/yashrajshuklaaa/homeostat/api/v1alpha1"
)

// HomeostatRecommendationReconciler watches HomeostatRecommendation objects.
//
// Policy enforcement happens before this reconciler ever sees an object:
// our Kyverno ClusterPolicy runs in Enforce mode with background: false,
// which validates synchronously at admission time. A recommendation that
// violates a guardrail never reaches etcd - the create request is rejected
// outright. So by the time Reconcile observes an object, it has already
// passed policy. See docs/adr/0001-recommendation-as-crd.md for the full
// reasoning.
type HomeostatRecommendationReconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=homeostat.dev,resources=homeostatrecommendations,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=homeostat.dev,resources=homeostatrecommendations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

func (r *HomeostatRecommendationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var rec homeostatv1alpha1.HomeostatRecommendation
	if err := r.Get(ctx, req.NamespacedName, &rec); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("fetching recommendation: %w", err)
	}

	switch rec.Status.Phase {
	case homeostatv1alpha1.PhaseAdmitted:
		logger.Info("applying admitted recommendation", "target", rec.Spec.Target.Name, "agent", rec.Spec.AgentName)
		return r.applyToTarget(ctx, &rec)

	case homeostatv1alpha1.PhaseBlocked:
		// Kept for forward compatibility: a future background/audit-mode
		// policy could conceivably flag an already-created object as
		// Blocked after the fact, even though our current Enforce-mode
		// policy never allows that state to occur today.
		logger.Info("recommendation blocked, skipping", "target", rec.Spec.Target.Name, "reason", rec.Status.Message)
		return ctrl.Result{}, nil

	case homeostatv1alpha1.PhaseApplied, homeostatv1alpha1.PhaseFailed:
		// Terminal states, nothing left to do.
		return ctrl.Result{}, nil

	default:
		// Pending / unset. The object's existence already means it passed
		// Kyverno's Enforce-mode admission check - see the type comment
		// above. Mark it Admitted and requeue so the Admitted branch
		// picks it up on the next reconcile.
		logger.Info("recommendation passed admission, marking Admitted", "target", rec.Spec.Target.Name)
		rec.Status.Phase = homeostatv1alpha1.PhaseAdmitted
		rec.Status.Message = "admitted at creation time by Kyverno Enforce policy"
		if err := r.Status().Update(ctx, &rec); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating status to Admitted: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}
}

// applyToTarget patches the target Deployment's container resource requests
// with the admitted recommendation's proposed values, then flips the
// recommendation's phase to Applied or Failed.
//
// NOTE: patches the Deployment directly for now rather than going through
// VPA. Good enough to prove the pipeline end to end; VPA integration is
// tracked on the roadmap.
func (r *HomeostatRecommendationReconciler) applyToTarget(
	ctx context.Context,
	rec *homeostatv1alpha1.HomeostatRecommendation,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if rec.Spec.Target.Kind != "Deployment" {
		return r.markFailed(ctx, rec, fmt.Sprintf("unsupported target kind %q, only Deployment is supported today", rec.Spec.Target.Kind))
	}

	var dep appsv1.Deployment
	key := types.NamespacedName{Name: rec.Spec.Target.Name, Namespace: rec.Spec.Target.Namespace}
	if err := r.Get(ctx, key, &dep); err != nil {
		return r.markFailed(ctx, rec, fmt.Sprintf("fetching target deployment: %v", err))
	}

	containers := dep.Spec.Template.Spec.Containers
	applied := 0
	for _, delta := range rec.Spec.ResourceDeltas {
		for i := range containers {
			if containers[i].Name != delta.Container {
				continue
			}
			if containers[i].Resources.Requests == nil {
				containers[i].Resources.Requests = corev1.ResourceList{}
			}
			for name, qty := range delta.ProposedRequests {
				containers[i].Resources.Requests[name] = qty
			}
			applied++
		}
	}

	if applied == 0 {
		return r.markFailed(ctx, rec, "no matching containers found on target deployment")
	}

	if err := r.Update(ctx, &dep); err != nil {
		return r.markFailed(ctx, rec, fmt.Sprintf("patching deployment: %v", err))
	}

	logger.Info("applied recommendation", "target", rec.Spec.Target.Name, "containersUpdated", applied)
	rec.Status.Phase = homeostatv1alpha1.PhaseApplied
	rec.Status.Message = fmt.Sprintf("patched %d container(s)", applied)
	if err := r.Status().Update(ctx, rec); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status to Applied: %w", err)
	}

	return ctrl.Result{}, nil
}

// markFailed sets the recommendation's phase to Failed with the given
// message and persists it.
func (r *HomeostatRecommendationReconciler) markFailed(
	ctx context.Context,
	rec *homeostatv1alpha1.HomeostatRecommendation,
	msg string,
) (ctrl.Result, error) {
	rec.Status.Phase = homeostatv1alpha1.PhaseFailed
	rec.Status.Message = msg
	if err := r.Status().Update(ctx, rec); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status to Failed: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *HomeostatRecommendationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&homeostatv1alpha1.HomeostatRecommendation{}).
		Complete(r)
}
