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
// It doesn't decide whether a recommendation is safe - Kyverno does that at
// admission time and writes the result to status.phase. This reconciler
// just reacts to that decision: apply if Admitted, do nothing if Blocked.
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
		logger.Info("recommendation blocked, skipping", "target", rec.Spec.Target.Name, "reason", rec.Status.Message)
		return ctrl.Result{}, nil

	default:
		// Pending / unset - admission hasn't run yet, nothing to do.
		return ctrl.Result{}, nil
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
