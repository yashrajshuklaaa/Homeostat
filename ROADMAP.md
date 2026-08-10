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
