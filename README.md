# EASL

**Epistemic Assumption State Layer** — a small, domain-neutral Go primitive for turning evidence, freshness, contradictions, and assumption dependencies into a deterministic epistemic state.

EASL does **not** execute actions and does **not** decide policy. It answers a narrower question:

> What can the system currently justify from the evidence it has?

A downstream policy engine such as Aegis-EGE can consume that result and decide whether an execution intent should be `ALLOW`, `BLOCK`, or `ESCALATE`.

```text
DoctrineManifest / active doctrine
       |
       v
GenesisManifest + external verification
       |
       v
EASL Bootstrap
  BOOTSTRAP_READY
       |
       v
  EASL Runtime
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

Operational evaluation is Genesis-gated. A caller must first verify the Level -1
Genesis package and obtain an EASL `Runtime`:

```go
runtime, genesisResult := easl.Bootstrap(ctx, easl.BootstrapInput{
    Manifest: manifest,
    Verification: genesis.Context{
        Now:                          time.Now().UTC(),
        MinimumAcceptedEpoch:         minimumEpoch,
        MinimumAcceptedDoctrineEpoch: minimumDoctrineEpoch,
        ExpectedDoctrineManifestHash: activeDoctrineDigest,
        ExpectedImplementationDigest: runtimeDigest,
        RequiredConformance:          genesis.ConformanceC3,
    },
    Verifier: verifier,
})
if genesisResult.State != genesis.StateReady {
    // fail closed: runtime is nil
    return
}

result, err := runtime.Evaluate(easl.Snapshot{
    At: time.Now().UTC(),
    Evidence: []easl.Evidence{{
        ID:         "deployment-health",
        ObservedAt: observedAt,
        ExpiresAt:  &expiresAt,
    }},
    Assumptions: []easl.Assumption{{
        ID:       "target-is-healthy",
        Requires: []easl.EvidenceID{"deployment-health"},
    }},
})
```

The legacy package-level `easl.Evaluate(...)` entry point is intentionally
fail-closed. It returns `ErrGenesisNotReady` and never performs operational
evaluation. A zero-value `Runtime` behaves the same way.

This enforces:

```text
NO_BOOTSTRAP_WITHOUT_VALID_GENESIS

GenesisManifest
      |
      v
genesis.Verify
      |
      +-- GENESIS_LOCKED --> no Runtime --> evaluation denied
      |
      '-- BOOTSTRAP_READY --> Runtime.Evaluate
```

The operational bootstrap additionally requires an explicit expected Level -2 doctrine manifest digest, implementation digest, and minimum conformance level. The doctrine epoch may also be fenced against a monotonic constitutional minimum. This prevents accidentally accepting a
manifest that is not bound to the runtime artifact being started.

The evaluation result remains machine-readable:

```json
{
  "state": "VALID",
  "evidence_status": "SUFFICIENT"
}
```

When evidence is stale, contradicted, missing, an assumption expires, its bound
subject state changes, or an upstream assumption becomes invalid, EASL returns
the affected evidence and assumptions explicitly.

Subject-state bindings remain domain-neutral opaque tokens. Producers decide
how to canonicalize a subject state; EASL only checks whether the state that
justified an assumption still matches the currently observed state:

```go
result, err := runtime.Evaluate(easl.Snapshot{
    At: time.Now().UTC(),
    StateBindings: []easl.StateBinding{{
        ID:       "target-state",
        Expected: authorizedStateDigest,
        Observed: currentStateDigest,
    }},
    Assumptions: []easl.Assumption{{
        ID:                    "safe-to-act",
        RequiresStateBindings: []easl.StateBindingID{"target-state"},
    }},
})
```

If the opaque tokens differ, the assumption is invalidated with
`SUBJECT_STATE_CHANGED`.

## Conformance

EASL now publishes machine-readable conformance vectors for the subject-state binding primitive in:

`conformance/subject_state_binding.json`

The Go implementation executes these vectors in CI. Other language consumers may use the same vectors to verify compatibility without copying domain semantics into EASL.

This does **not** make every compatibility layer an independent full EASL implementation. The Go module remains the reference implementation, and new conformance surfaces should be added only after a primitive is consumed outside the reference runtime.

## Semantics in v0

- `Snapshot.At` makes freshness evaluation deterministic and replayable.
- Required evidence must exist and remain fresh.
- Fresh evidence may explicitly contradict assumptions.
- Assumptions may carry an explicit `ValidUntil`; expiry invalidates the assumption deterministically at the snapshot time.
- Assumptions may require opaque subject-state bindings; a changed binding invalidates the old justification with `SUBJECT_STATE_CHANGED`.
- Assumption invalidation propagates through `DependsOn` edges.
- Missing required evidence is a normal `INSUFFICIENT` state; malformed assumption references, duplicate IDs, and dependency cycles return errors so callers can fail closed.
- EASL does not infer domain meaning, mutate external state, or contain an execution policy engine.

## Status

Early core contract. The next capabilities should be extracted only when real consumers require them. Architecture follows evidence.

Subject-state binding is now an **earned primitive**: it has independent consumers in Aegis-EGE and CI Retry Gate, shared conformance vectors, and a live external mutation proof. See [Subject-State Binding Consumer Proof](docs/subject-state-binding-consumer-proof.md).

## Doctrine

The reasoning model behind EASL is documented in:

- [Assumption Decay Doctrine](docs/assumption-decay-doctrine.md)
- [Assumption Lifecycle](docs/assumption-lifecycle.md)
- [Invalidation Semantics](docs/invalidation-semantics.md)
- [Temporal Validity](docs/temporal-validity.md)

The doctrine is a specification/theory layer inside EASL. It should only become a separate repository if it later develops an independent lifecycle with versioning, conformance tests, and multiple implementations.

