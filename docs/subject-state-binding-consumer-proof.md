# Subject-State Binding Consumer Proof

Status: **earned primitive**

This note records why subject-state binding exists in EASL and the independent consumer evidence that justifies keeping it as a domain-neutral primitive.

## Invariant

> A justification bound to subject state `S` must not remain valid when the observed subject state is no longer `S`.

EASL represents that invariant with an opaque `StateBinding`:

- `Expected`: the state that justified the assumption;
- `Observed`: the state seen at evaluation time;
- mismatch: `SUBJECT_STATE_CHANGED`.

EASL does not interpret the token. Canonicalization and domain meaning remain with the producer or consumer.

## Consumer 1 — Aegis-EGE

Repository: `achirothmane/aegis-ege`

Reference main commit:

`32cc52cb9e5276d7c160f100d9270d2acd4e8fe8`

Aegis consumes EASL subject-state bindings for execution authorization. In the Kubernetes execution path, authorization is bound to state including:

- resource version;
- execution plan digest.

A mismatch is detected by EASL as `SUBJECT_STATE_CHANGED`, while Aegis maps the generic invalidation back to its operational policy reasons such as `ResourceVersionChanged` and `ExecutionPlanChanged`.

This proves that the primitive can sit below execution-policy semantics without absorbing Kubernetes or authorization policy into EASL.

## Consumer 2 — CI Retry Gate

Repository: `achirothmane/workflow-failure-lab`

Reference main commit:

`0297cefd4b47f2073cada67cf25c263feb1b2b8a`

CI Retry Gate consumes the same invariant through a deliberately narrow Python compatibility layer.

The legacy full-workflow rerun path binds its justification to:

- repository;
- run ID;
- run attempt;
- head SHA;
- workflow ID;
- failed-job set.

The selective job-rerun path binds its justification to:

- run attempt;
- head SHA;
- workflow ID;
- workflow lifecycle;
- exact failed job execution.

A changed binding fails closed before mutation.

The Python consumer is tested against the same machine-readable conformance vectors published by EASL.

## Live external mutation proof

External consumer repository:

`achirothmane/ci-retry-gate-consumer-e2e`

Reference main commit:

`1b3a1c102cbba4652e13d2fb97bc38cf880f71c1`

Successful live gate run:

`36323452061`

The controlled GitHub Actions experiment produced two independently eligible transient failures. The gate then proved:

1. two jobs were safe selective candidates;
2. exactly one rerun mutation was requested from the evaluated state epoch;
3. GitHub created workflow attempt 2;
4. exactly one failed job received a new execution timestamp;
5. the other failed job was copied forward unchanged by GitHub;
6. the completed second attempt was re-evaluated;
7. the configured attempt boundary prevented any second mutation.

This is a real external mutation proof, not only an in-process unit test.

## Why this primitive remains in EASL

The same epistemic rule now appears in two materially different domains:

- infrastructure execution authorization;
- CI rerun authorization.

The domain-specific policy and operational reason codes differ, but the lower-level invalidation rule is the same.

That is the evidence threshold for extraction:

`consumer pressure → duplicated generic rule → domain-neutral primitive → cross-domain proof`

Subject-state binding therefore qualifies as an earned EASL primitive rather than speculative architecture.

## What this evidence does not justify

This proof does **not** justify:

- moving authorization policy into EASL;
- interpreting resource versions, Git SHAs, job identities, or plan digests inside EASL;
- adding persistence, refresh orchestration, confidence scoring, or automatic mutation;
- building a second full EASL implementation in Python;
- extracting a standalone specification repository solely because one primitive now has conformance vectors.

Further architecture still requires independent consumer pressure.
