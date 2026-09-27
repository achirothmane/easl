# Assumption Decay Doctrine

> Status: draft doctrine for EASL. This document defines the reasoning model that the implementation should converge toward. It is not an execution-policy specification.

## 1. Purpose

Operational systems act on assumptions.

Examples:

- the target is healthy;
- a dependency still has the same identity;
- a safety precondition remains true;
- a policy input is still current;
- an observation still describes the world closely enough to justify a decision.

Those assumptions are not permanently true merely because they were once justified.

The **Assumption Decay Doctrine** defines how EASL should reason about the loss of epistemic validity over time and through dependency change.

The doctrine exists to answer one question:

> **What can the system still justify now, from the evidence and dependencies that were used to justify it before?**

It deliberately does **not** answer:

> **What action should the system take?**

That second question belongs to downstream policy and execution systems such as Aegis-EGE.

---

## 2. Core principle

### Assumptions are leased, not owned

An assumption should be treated as a temporary lease on a claim about the world.

Its validity lasts only while the conditions that justified it remain satisfied.

A previous evaluation is therefore not a permanent fact. It is a claim with an epistemic lifetime.

For an assumption A, validity at a later time cannot be inferred merely from validity at an earlier time:

    Valid(A, t0) = true

does not imply:

    Valid(A, t) = true

Validity at time t must be re-established from the state of its supporting evidence, explicit temporal boundary, subject-state bindings, contradictions, and dependency graph.

---

## 3. Decay is not the same as falsehood

EASL must keep several concepts separate.

### 3.1 Expired

The system once had enough justification for the assumption, but the declared validity window has ended.

Example:

    evaluated_at = 12:00:00
    valid_until  = 12:00:05
    now          = 12:00:05

At the exact boundary, the assumption is expired.

Expiry says:

> The previous justification is no longer current enough.

It does not necessarily say:

> The assumption is false.

### 3.2 Contradicted

Fresh evidence explicitly conflicts with an assumption.

Contradiction is stronger than expiry because the system possesses active evidence against the assumption.

### 3.3 Unsupported

Required evidence is missing or no longer usable.

This means the assumption cannot currently be justified from its declared evidence basis.

### 3.4 Dependency-invalid

An assumption depends on another assumption that is no longer valid.

The invalidity propagates even when no direct evidence about the downstream assumption has changed.

### 3.5 Subject-state changed

A justification may be bound to an opaque representation of the subject state that was observed when the assumption became usable.

If the currently observed state token no longer matches the token to which the assumption was bound, the old justification is no longer usable.

This says:

> The assumption was justified for a different subject state.

It does not require EASL to understand what the token means.

Kubernetes resource versions, CI run/head state, database revisions, or composite digests remain domain concerns. Producers canonicalize them into opaque state-binding tokens.

### 3.6 Structurally indeterminate

The system cannot evaluate the assumption graph safely because the input is malformed, cyclic, references unknown assumptions, or otherwise violates the contract.

This is not evidence that the assumption is false. It means the epistemic state cannot be established reliably.

Downstream consumers should decide how such uncertainty affects execution policy.

---

## 4. Decay is multi-causal

Assumption decay is not only a clock.

The effective validity of an assumption is constrained by several independent mechanisms:

    Evidence freshness
            +
    Explicit assumption lifetime
            +
    Contradiction edges
            +
    Dependency validity
            +
    Subject-state binding
            +
    Structural evaluability
            ↓
    Current epistemic state

A robust system should not compress all of these into one opaque score.

The cause matters because downstream systems may treat different forms of invalidity differently.

For example:

    expired assumption       -> request revalidation
    contradicted assumption  -> hard block
    missing evidence         -> gather more evidence
    dependency invalid       -> recompute dependent state
    subject state changed    -> revalidate against current state
    malformed graph          -> fail closed / escalate

Those mappings are **consumer policy**, not EASL policy.

---

## 5. No hidden confidence score by default

EASL should prefer explicit, inspectable validity conditions over an unexplained confidence number.

A statement such as:

    confidence = 0.63

is weaker operational evidence than:

    assumption = target-is-healthy
    valid_until = 2026-09-27T02:45:05Z
    required_evidence = [node-health]
    dependency = network-path-valid
    invalidation_reason = STALE_EVIDENCE

Numeric confidence or probabilistic decay may later be useful for specific domains, but it must not replace causal evidence and explicit invalidation semantics.

If probabilistic models are added later, they should be evidence producers or optional epistemic inputs, not a hidden global truth mechanism.

---

## 6. Temporal validity must be explicit

Time-sensitive assumptions require an explicit validity boundary.

Current EASL expresses this with Assumption.ValidUntil.

The rule is:

    expired(A, t) iff t >= ValidUntil(A)

The boundary is intentionally closed on expiry:

    now < valid_until   -> still temporally valid
    now >= valid_until  -> expired

This makes replay deterministic and removes ambiguity at the exact boundary.

### Why absolute validity beats hidden age checks

A consumer may derive the validity boundary from policy:

    critical action -> 5 seconds
    high action     -> 10 seconds
    low action      -> 60 seconds

But EASL should receive the resulting epistemic fact:

    valid_until = evaluated_at + selected_max_age

rather than owning the action-risk policy itself.

This preserves the boundary:

    Consumer policy chooses the lifetime.
    EASL evaluates whether that lifetime has ended.

---

## 7. Evidence and assumptions decay independently

Evidence freshness and assumption lifetime are related but different.

An evidence item may remain fresh while an assumption that depended on it expires because the assumption has a stricter validity window.

Conversely, an assumption may have a long nominal lifetime but become invalid immediately because required evidence expires or is contradicted.

Therefore:

    Validity(A) != Freshness(E)

and:

    ValidUntil(A) != ExpiresAt(E)

unless a producer or consumer explicitly chooses to make them equal.

EASL should preserve both dimensions.

---

## 8. Dependency decay must propagate

Assumptions form a directed dependency graph.

If:

    A depends on B
    B depends on C

and C becomes invalid, then B and A can no longer remain justified merely because their own local evidence did not change.

Conceptually:

    Invalid(C) => Invalid(B) => Invalid(A)

EASL therefore propagates invalidity through DependsOn.

This propagation should remain causal and inspectable.

A downstream consumer should be able to answer:

    Why is A invalid?
    Because B became invalid.
    Why is B invalid?
    Because C expired.

The system should preserve enough provenance to reconstruct this chain.

---

## 9. Revalidation renews justification; it does not rewrite history

When an expired or invalid assumption is re-evaluated successfully, the new justification should be treated as a new epistemic state.

Revalidation must not imply that the previous expired state never existed.

A useful lifecycle is:

    justified
       ↓
    valid
       ↓
    decayed / contradicted / unsupported
       ↓
    invalid
       ↓
    re-evaluated
       ↓
    new justification

For audit-sensitive systems, the old and new evaluations should be distinguishable by time, evidence set, dependency state, and resulting validity window.

This doctrine therefore favors **appendable epistemic history** over silent mutation of truth.

The current EASL core does not yet implement an event log or historical store. That capability should only be added when real consumers require it.

---

## 10. Decay should be deterministic when inputs are deterministic

Given the same:

- evaluation time;
- evidence set;
- evidence expiry boundaries;
- contradiction edges;
- assumption graph;
- assumption validity boundaries;
- opaque subject-state bindings;

EASL should return the same epistemic evaluation.

This is why Snapshot.At is supplied by the caller instead of reading the wall clock internally.

Determinism enables replay, testing, incident reconstruction, conformance testing, comparison between implementations, and reproducible policy decisions downstream.

---

## 11. Decay must not silently upgrade authority

An epistemic evaluation can reduce what a consumer may safely justify.

It should not silently grant execution authority.

EASL may conclude:

    VALID

but that does not mean:

    ALLOW

The distinction is fundamental:

    EASL:
    "What is justified?"

    Aegis-EGE:
    "Given what is justified, what is permitted?"

A valid assumption can still lead to a blocked action because of blast radius, authorization, state binding, policy, identity, rate limits, or other execution constraints.

---

## 12. The doctrine is domain-neutral

The decay model should not depend on Kubernetes, CI, databases, cloud APIs, or AI agents.

Domain systems decide:

- what an assumption means;
- what evidence supports it;
- what evidence contradicts it;
- how long it may remain valid;
- which dependencies it has.

EASL evaluates the declared epistemic structure.

The intended layering is:

    Domain evidence producers
            ↓
    EASL
      evidence
      freshness
      assumption lifetime
      contradiction
      subject-state binding
      dependency invalidation
            ↓
    Domain policy / governance
            ↓
    Action

---

## 13. Current normative invariants

The following invariants should remain stable unless evidence from real consumers forces a revision.

### Invariant 1 — no permanent validity by implication

An assumption is not permanently valid merely because it was once supported.

### Invariant 2 — exact expiry boundary is invalid

If Snapshot.At equals Assumption.ValidUntil, the assumption is expired.

### Invariant 3 — fresh contradiction dominates support

Fresh evidence that explicitly contradicts an assumption invalidates it even when other evidence remains available.

### Invariant 4 — dependency invalidity propagates

A dependent assumption cannot remain valid when a declared upstream assumption is invalid.

### Invariant 5 — missing support is not contradiction

Missing evidence means insufficient justification, not evidence of the opposite claim.

### Invariant 6 — malformed epistemic structure is not silently accepted

Unknown assumption references, duplicate identifiers, or dependency cycles must not produce a valid state.

### Invariant 7 — consumer policy remains outside EASL

EASL does not decide ALLOW, BLOCK, or ESCALATE.

### Invariant 8 — replay time is explicit

Temporal evaluation uses caller-supplied Snapshot.At, never hidden wall-clock time.

### Invariant 9 — justification is state-bound when declared

If an assumption requires a subject-state binding, the expected and currently observed opaque tokens must match. A mismatch invalidates the old justification rather than silently carrying it across a changed world state.

---

## 14. What the current implementation supports

The current EASL core implements:

    Evidence.ExpiresAt
    Assumption.ValidUntil
    Evidence -> explicit contradiction edges
    Assumption.Requires
    StateBinding.Expected / StateBinding.Observed
    Assumption.RequiresStateBindings
    Assumption.DependsOn
    dependency invalidation propagation
    deterministic Snapshot.At evaluation

Current aggregate states are:

    VALID
    DEGRADED
    INVALID

Current evidence summaries are:

    SUFFICIENT
    INSUFFICIENT
    CONTRADICTORY

Current invalidation causes include:

    MISSING_EVIDENCE
    STALE_EVIDENCE
    CONTRADICTED
    DEPENDENCY_INVALID
    ASSUMPTION_EXPIRED
    SUBJECT_STATE_CHANGED

The doctrine is intentionally deeper than the current implementation, but additions should be earned by concrete consumers rather than implemented speculatively.

---

## 15. What is deliberately not specified yet

The doctrine does not currently define:

- probabilistic half-life functions;
- confidence aggregation;
- Bayesian updates;
- source reputation scoring;
- trust-weighted evidence voting;
- persistence format;
- distributed consensus over epistemic state;
- cross-process event journals;
- automatic revalidation scheduling;
- domain-specific TTL values;
- execution policy.

These may become separate primitives later if real use demonstrates the need.

Until then:

> **Do not convert theoretical completeness into architectural surface area.**

---

## 16. Extraction rule

A new doctrine concept should become executable EASL code only when at least one real consumer requires deterministic behavior for it.

The preferred sequence is:

    Observed consumer need
            ↓
    document the epistemic rule
            ↓
    define a minimal invariant
            ↓
    add conformance-like tests
            ↓
    implement the smallest primitive
            ↓
    consume it in a real system

This keeps the architecture evidence-gated.

---

## 17. When this doctrine deserves its own repository

The Assumption Decay Doctrine should remain inside EASL until it develops an independent lifecycle.

A separate specification repository becomes justified when several of the following are true:

- more than one independent implementation exists;
- multiple systems consume the specification without depending on EASL;
- the doctrine has explicit versioning;
- conformance tests exist independently of the Go implementation;
- compatibility rules matter across versions;
- external contributors need to implement the semantics in other languages;
- the specification changes on a cadence distinct from EASL.

At that point the architecture may become:

    assumption-decay-doctrine   specification
              ↓
    easl                        reference implementation
              ↓
    aegis-ege                   governance / execution consumer

Until then, keeping the doctrine in easl/docs/ prevents premature fragmentation.

---

## 18. Governing sentence

The doctrine can be summarized in one rule:

> **An assumption remains usable only while the evidence, time boundary, bound subject state, and dependency structure that justify it remain valid at the moment it is consumed.**

Or, more compactly:

> **Justification decays; authority must be re-earned from current evidence.**
