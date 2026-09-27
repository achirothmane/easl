# EASL

**Epistemic Assumption State Layer** — a small, domain-neutral Go primitive for turning evidence, freshness, contradictions, and assumption dependencies into a deterministic epistemic state.

EASL does **not** execute actions and does **not** decide policy. It answers a narrower question:

> What can the system currently justify from the evidence it has?

A downstream policy engine such as Aegis-EGE can consume that result and decide whether an execution intent should be `ALLOW`, `BLOCK`, or `ESCALATE`.

```text
Evidence Producers
       |
       v
      EASL
  epistemic state
       |
       v
   Aegis-EGE
 policy + decision
       |
       v
ALLOW / BLOCK / ESCALATE
```

## Core contract

```go
result, err := easl.Evaluate(easl.Snapshot{
    At: time.Now(),
    Evidence: []easl.Evidence{
        {
            ID:         "deployment-health",
            ObservedAt: observedAt,
            ExpiresAt:  &expiresAt,
        },
    },
    Assumptions: []easl.Assumption{
        {
            ID:       "target-is-healthy",
            Requires: []easl.EvidenceID{"deployment-health"},
        },
    },
})
```

The result is machine-readable:

```json
{
  "state": "VALID",
  "evidence_status": "SUFFICIENT"
}
```

When evidence is stale, contradicted, missing, an assumption expires, or an upstream assumption becomes invalid, EASL returns the affected evidence and assumptions explicitly.

## Semantics in v0

- `Snapshot.At` makes freshness evaluation deterministic and replayable.
- Required evidence must exist and remain fresh.
- Fresh evidence may explicitly contradict assumptions.
- Assumptions may carry an explicit `ValidUntil`; expiry invalidates the assumption deterministically at the snapshot time.
- Assumption invalidation propagates through `DependsOn` edges.
- Missing required evidence is a normal `INSUFFICIENT` state; malformed assumption references, duplicate IDs, and dependency cycles return errors so callers can fail closed.
- EASL does not infer domain meaning, mutate external state, or contain an execution policy engine.

## Status

Early core contract. The next capabilities should be extracted only when real consumers require them. Architecture follows evidence.
