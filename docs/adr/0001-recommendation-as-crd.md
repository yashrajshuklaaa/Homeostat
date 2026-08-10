# ADR 0001: Recommendations are staged as a CRD, not applied directly

## Status
Accepted (implementation in progress - see "Open question" below)

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

## Open question

Kyverno's ValidatingPolicy runs at admission time (on create/update of the
HomeostatRecommendation itself) - it can reject the object outright, but
today nothing writes the *admitted* case back onto `status.phase`. Our
reconciler currently assumes something else already set `Admitted` or
`Blocked`.

Two ways to close this gap, still being evaluated:

1. A small mutating/validating webhook that runs after Kyverno, translating
   "admission succeeded" into `status.phase = Admitted` on create.
2. Skip Kyverno's admission-time blocking model entirely and instead have
   our own controller call out to Kyverno's policy engine as a library at
   reconcile time, setting status itself based on the result.

Leaning toward (1) since it keeps Kyverno as the actual policy authority
rather than duplicating its logic in our controller. Tracked as the next
implementation task.

## Consequences

- One extra hop (propose -> validate -> apply) versus a direct patch.
  Slightly more latency, meaningfully more safety and auditability.
- The controller must handle a Blocked recommendation gracefully (no-op,
  not a crash) - this is already covered by
  `TestReconcile_BlockedDoesNothing`.
