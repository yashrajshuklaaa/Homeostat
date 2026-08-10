# ADR 0001: Recommendations are staged as a CRD, not applied directly

## Status
Accepted

## Context

kagent agents could, in principle, patch a Deployment's resource requests
directly using their k8s_apply_manifest tool. That would be simpler than
what this repo does. We chose not to do that.

## Decision

Every optimization an agent proposes is written as a `HomeostatRecommendation`
object instead of a direct mutation. Kyverno validates the object against
guardrail policies (delta limits, time windows, label requirements). Only
after a recommendation is admitted does our controller patch the actual
workload.

## Why

- **Auditability.** A HomeostatRecommendation is a permanent record: which
  agent proposed it, why, and what the exact before/after values were.
  A direct patch leaves no equivalent trail beyond a generic Kubernetes
  event.
- **Policy enforcement needs something to inspect.** Kyverno validates
  objects at admission time. It can't inspect "an agent is about to call
  kubectl patch" - it needs a real object with fields to check
  (`deltaPercent`, `requiresLabels`, etc). Staging the proposal as its own
  resource is what makes policy enforcement possible at all.
- **Decouples proposing from applying.** The agent's job is narrow:
  look at metrics, propose a number. It doesn't need write access to
  Deployments, HPAs, or VPAs - only to HomeostatRecommendation objects.
  That's a meaningfully smaller blast radius if the agent misbehaves.

## How admission reaches the controller

Our Kyverno ClusterPolicy (`policies/max-rightsize-delta.yaml`) runs with
`validationFailureAction: Enforce` and `background: false`. That means
validation happens synchronously at admission time, before the object is
persisted to etcd. A recommendation that violates a guardrail is rejected
at the API server - it never becomes a stored object.

This has a useful consequence: the reconciler doesn't need a webhook or any
other mechanism to learn "did this pass policy?" - if `Reconcile` observes
an object at all, it already passed. On first reconcile of a freshly
created recommendation (phase unset), the controller marks it `Admitted`
directly and requeues, rather than waiting on an external signal.

This was originally left as an open question (see prior revision of this
ADR) while we considered building an admission webhook to bridge Kyverno's
decision into `status.phase`. That turned out to be unnecessary complexity
- Enforce-mode Kyverno already gates creation synchronously, so existence
is proof of admission.

The `Blocked` phase is retained in the type for forward compatibility, in
case a future policy runs in Audit/background mode and needs to flag an
already-created object after the fact - but no current policy produces
that state.

## Consequences

- One extra hop (propose -> validate -> apply) versus a direct patch.
  Slightly more latency, meaningfully more safety and auditability.
- No webhook infrastructure needed - fewer moving parts, no cert
  management, one less thing to operate.
- The `Blocked` phase exists but is currently unreachable in practice given
  our only policy runs in Enforce mode. `TestReconcile_BlockedDoesNothing`
  still covers it directly at the reconciler level, independent of how a
  future policy might set it.
