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
real Kyverno and real kagent installed together. That's the next milestone,
tracked below.

## Next (short-term, actively planned)

- [ ] First real end-to-end run on a local (kind) cluster: real Kyverno,
      real CRD, controller against a live API server, a hand-crafted test
      recommendation pushed through propose -> admit -> apply
- [ ] Second Kyverno policy: business-hours change freeze
- [ ] `require-pdb` policy - block rightsizing on workloads without a
      PodDisruptionBudget
- [ ] Example manifests under `examples/` so the pipeline is demoable
      without kagent running
- [ ] VPA integration as an alternative to direct Deployment patching

## Vision (not built, not scheduled - direction only)

Longer-term ideas from early product framing. None of this exists in code
yet; listed here so ambition and reality don't get confused.

- Dry-run mode with estimated savings shown before any change is applied
- A human-approval step for recommendations (currently: Kyverno's Enforce
  policy is the only gate - see [ADR 0001](docs/adr/0001-recommendation-as-crd.md))
- Cost/savings reporting for FinOps stakeholders
- Slack/notification integration
- Proactive scale-up under detected memory pressure (today: the agent only
  proposes what current usage data supports, no predictive behavior)
- Namespace-level cost-governance policies (e.g. spot-instance requirements)

## How to read this file

- **Built** = you can `go test ./...` or read the code and see it work today.
- **Next** = actively being worked on, order roughly reflects priority.
- **Vision** = direction, not commitment. Items move to Next when there's
  an actual design, not just an idea.
