package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	homeostatv1alpha1 "github.com/YOUR_USERNAME/homeostat/api/v1alpha1"
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
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=autoscaling.k8s.io,resources=verticalpodautoscalers,verbs=get;list;watch;update;patch

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

// applyToTarget patches the target HPA/VPA with the admitted values.
// TODO: fetch the VPA/HPA referenced by rec.Spec.Target, apply
// ResourceDeltas, flip status to Applied or Failed.
func (r *HomeostatRecommendationReconciler) applyToTarget(
	ctx context.Context,
	rec *homeostatv1alpha1.HomeostatRecommendation,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *HomeostatRecommendationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&homeostatv1alpha1.HomeostatRecommendation{}).
		Complete(r)
}
