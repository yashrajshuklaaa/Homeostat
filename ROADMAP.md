# Roadmap

This splits what Homeostat actually does today from what it's meant to
become. If you're evaluating this repo, "Built" is the only section that's
verified working. Everything else is a plan, not a claim.

## Built (verified, tested, in main)

- `HomeostatRecommendation` CRD - the staged-proposal object agents write to
- Reconciler that watches recommendations and patches the target Deployment
  once admitted ([internal/controller](internal/controller))
- One Kyverno guardrail: `max-rightsize-delta` blocks any proposed change
  over 30% ([policies/max-rightsize-delta.yaml](policies/max-rightsize-delta.yaml))
- kagent Agent + ModelConfig spec describing an optimization agent
  ([agents/optimization-agent.yaml](agents/optimization-agent.yaml))
- Unit tests covering all reconciler phase branches
- CI (build, vet, test, policy lint)

**Not yet verified:** none of the above has run against a real cluster with
real Kyverno and real kagent installed together. That's the immediate next
milestone, first item below.

## Next (short-term, actively planned)

- [ ] First real end-to-end run on a local (kind) cluster: real Kyverno,
      real CRD, controller against a live API server, a hand-crafted test
      recommendation pushed through propose -> admit -> apply
- [ ] Kubernetes Events on the reconciler (`recordEvent`) - so
      `kubectl get events` shows *why* a recommendation was applied or
      failed, not just a status string nobody's watching
- [ ] Status Conditions (`Validated`, `Applied`, `RolledBack`) alongside
      `status.phase` - standard `kubectl describe` UX, makes state
      inspectable without reading controller logs
- [ ] Health/liveness probes on the manager (`healthz`/`readyz`) -
      required before this can run as a real Deployment in a cluster,
      not just `make run` on a laptop
- [ ] Leader election - required before running more than one controller
      replica; without it, HA setups split-brain
- [ ] Second Kyverno policy: business-hours change freeze
- [ ] `require-pdb` policy - block rightsizing on workloads without a
      PodDisruptionBudget
- [ ] Example manifests under `examples/` so the pipeline is demoable
      without kagent running
- [ ] VPA integration as an alternative to direct Deployment patching

## Explicitly rejected (and why)

- **A Go-native "recommendation engine"** that computes rightsizing values
  with a fixed formula (e.g. `peak_usage * 1.2`). This was proposed at one
  point and deliberately not adopted: the whole point of this project is
  that the kagent agent does that reasoning - querying Prometheus,
  weighing context, writing a justification - not a hardcoded heuristic in
  Go. Duplicating that logic natively would make the kagent integration
  decorative instead of load-bearing, which defeats the actual thesis of
  the project. If a non-AI fallback mode is ever wanted (e.g. for local
  dev without kagent running), it needs to be clearly labeled as a
  fallback, not framed as "the engine."

## Vision (not built, not scheduled - direction only)

Longer-term ideas from early product framing. None of this exists in code
yet; listed here so ambition and reality don't get confused.

- Dry-run mode with estimated savings shown before any change is applied
- A human-approval step for recommendations (currently: Kyverno's Enforce
  policy is the only gate - see [ADR 0001](docs/adr/0001-recommendation-as-crd.md))
- Cost/savings reporting for FinOps stakeholders (would need real cost
  data, e.g. via OpenCost, not invented numbers)
- Slack/notification integration
- Automatic rollback if an applied change causes an SLO regression
- Background scanner that periodically asks the agent to evaluate all
  Deployments, instead of requiring someone to trigger it
- Namespace-level cost-governance policies (e.g. spot-instance requirements)
- Multi-cluster fleet support

## How to read this file

- **Built** = you can `go test ./...` or read the code and see it work today.
- **Next** = actively being worked on, order roughly reflects priority.
- **Explicitly rejected** = considered and deliberately not done, with the
  reasoning kept so it isn't re-proposed without context.
- **Vision** = direction, not commitment. Items move to Next when there's
  an actual design, not just an idea.

## Found during live testing (not yet fixed)

- The reconciler currently reprocesses the same recommendation multiple
  times per logical change (status updates trigger new watch events,
  which re-trigger Reconcile). Harmless today since apply is idempotent,
  but wasteful. Fix: check `status.observedGeneration` against
  `metadata.generation` at the top of Reconcile and skip if already
  processed - the field already exists on the type, just unused.

## Real gaps identified in review (worth fixing, in priority order)

These came from actually re-reading the architecture against what's built,
not from feature-comparison against other products. Each one is something
we designed for but never verified or finished.

1. **kagent has never actually produced a recommendation.** Every test so
   far used a hand-written CRD. The agent spec exists but the real loop -
   agent reasons over metrics, calls its own tool, writes a
   HomeostatRecommendation - has never run. This is the actual claim of
   the project and it's currently unverified. Highest priority.
2. **Direct Deployment patching should become VPA integration.** Patching
   a Deployment's containers directly restarts pods with no PDB-awareness.
   Writing to a VerticalPodAutoscaler instead lets VPA's mature eviction
   logic handle the actual rollout safely. This was the original intent,
   simplified for MVP speed - worth reinstating.
3. **No Kubernetes Events on the reconciler.** Cheap fix, real value:
   `kubectl get events` currently shows nothing, so debugging why a
   recommendation was applied or failed means reading controller logs.
4. **README overclaims "native autoscaling (HPA/VPA/Cluster Autoscaler)"
   integration that doesn't exist yet** - only direct Deployment patching
   is implemented today. Needs a two-line honesty fix independent of when
   #2 actually lands.

## Explicitly not adopted from external review

A review pass also proposed a metrics collector, a Go-native
recommendation engine, a background scanner, automatic rollback,
Prometheus savings dashboards, a Helm chart, and a competitive
positioning strategy against five funded companies - all framed as
immediate priorities.

None of that is adopted. It's not that these ideas are bad in the
abstract - some may become real Vision items later - but treating a
solo project's next five commits as "ship a CAST AI competitor" is how
small open-source tools die unfinished. The plan stays: small, honest,
one real gap at a time, verified on a real cluster before moving on.
