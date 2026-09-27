# Invalidation Semantics

> Status: draft specification companion to the [Assumption Decay Doctrine](assumption-decay-doctrine.md) and [Assumption Lifecycle](assumption-lifecycle.md).

This document defines what invalidation means in EASL, how invalidation causes are represented, how they compose, and how invalidity propagates through assumption dependencies.

It does **not** define execution policy.

---

## 1. Purpose

EASL should never reduce all invalidity to a generic boolean when the cause is known.

The difference between:

- evidence missing;
- evidence stale;
- an assumption expired;
- an assumption contradicted;
- an upstream dependency becoming invalid;

is operationally meaningful.

A consumer may react differently to each cause.

Therefore EASL invalidation is designed around a simple rule:

> **Invalidity must remain causal and inspectable.**

The current implementation exposes invalidation causes through:

    Evaluation.Invalidations

Each invalidation identifies:

- the affected assumption;
- a machine-readable reason;
- the evidence or dependency involved when applicable.

---

## 2. Invalidation is about justification

Invalidation does not necessarily mean that a proposition is objectively false.

It means:

> **EASL can no longer justify the assumption under the current snapshot.**

This distinction matters.

For example:

    MISSING_EVIDENCE

means:

> the declared support is unavailable.

It does not mean:

> the opposite claim has been proven.

Likewise:

    ASSUMPTION_EXPIRED

means:

> the previous justification has crossed its temporal validity boundary.

It does not mean:

> the world definitely changed.

Only explicit contradiction semantics represent active evidence against the assumption.

---

## 3. Current invalidation reasons

Current EASL invalidation reasons are:

    MISSING_EVIDENCE
    STALE_EVIDENCE
    CONTRADICTED
    DEPENDENCY_INVALID
    ASSUMPTION_EXPIRED

Each reason has distinct semantics.

---

## 4. MISSING_EVIDENCE

### Definition

An assumption declares an evidence dependency through:

    Assumption.Requires

If a required evidence ID is absent from the snapshot, EASL emits:

    MISSING_EVIDENCE

Example:

    assumption:
      id: safe-to-execute
      requires: [cluster-health]

    snapshot evidence:
      []

Result:

    safe-to-execute -> MISSING_EVIDENCE(cluster-health)

### Meaning

The assumption is unsupported because one of its declared evidence requirements cannot be satisfied.

This is an insufficiency condition.

It is not a contradiction.

### Invariant

For required evidence E:

    E not present
        =>
    MISSING_EVIDENCE

The evaluator should not silently ignore missing declared support.

---

## 5. STALE_EVIDENCE

### Definition

Evidence may declare an absolute freshness boundary:

    Evidence.ExpiresAt

At evaluation time:

    stale(E, t) iff t >= E.ExpiresAt

If stale evidence is required by an assumption, EASL emits:

    STALE_EVIDENCE

against that assumption.

### Meaning

The evidence existed, but its declared freshness window is no longer valid for the current snapshot.

The system cannot continue to rely on it as current support.

### Distinction from missing evidence

    missing:
      evidence does not exist in the snapshot

    stale:
      evidence exists but can no longer justify the assumption

Both can lead to insufficient justification, but they describe different failure modes.

### Distinction from assumption expiry

Evidence freshness and assumption validity are independent.

    Evidence.ExpiresAt
        !=
    Assumption.ValidUntil

unless a producer or consumer explicitly makes them equal.

---

## 6. CONTRADICTED

### Definition

Fresh evidence may explicitly declare that it contradicts one or more assumptions:

    Evidence.Contradicts

If the evidence is fresh and the target assumption exists, EASL emits:

    CONTRADICTED

Example:

    evidence:
      id: regression-signal
      contradicts: [safe-to-deploy]

Result:

    safe-to-deploy -> CONTRADICTED(regression-signal)

### Meaning

Contradiction is active evidence against the assumption.

It is semantically stronger than absence of support.

### Freshness condition

Stale contradictory evidence does not invalidate an assumption in the current implementation.

The contradiction edge is active only while the evidence itself is fresh.

Conceptually:

    Fresh(E)
    AND E contradicts A
        =>
    CONTRADICTED(A, E)

### Unknown contradiction target

If evidence claims to contradict an assumption that does not exist in the snapshot, the snapshot is structurally invalid.

EASL returns an evaluation error rather than creating a synthetic assumption or silently ignoring the reference.

---

## 7. ASSUMPTION_EXPIRED

### Definition

An assumption may declare:

    Assumption.ValidUntil

At evaluation time:

    expired(A, t) iff t >= A.ValidUntil

When the boundary is reached, EASL emits:

    ASSUMPTION_EXPIRED

### Meaning

The previous justification has exceeded its declared lifetime.

This is a temporal invalidation of the assumption itself.

### Boundary rule

The exact boundary is invalid:

    t < ValidUntil   -> not expired
    t = ValidUntil   -> expired
    t > ValidUntil   -> expired

This must remain deterministic.

### Policy boundary

EASL does not choose the lifetime.

A consumer may derive the lifetime from policy:

    critical -> 5 seconds
    high     -> 10 seconds
    low      -> 60 seconds

EASL evaluates only the resulting absolute validity boundary.

---

## 8. DEPENDENCY_INVALID

### Definition

Assumptions may depend on other assumptions:

    Assumption.DependsOn

If an upstream assumption becomes invalid, a dependent assumption can no longer remain justified merely because its own local evidence is unchanged.

EASL emits:

    DEPENDENCY_INVALID

Example:

    A depends on B
    B depends on C

If C expires:

    C -> ASSUMPTION_EXPIRED
    B -> DEPENDENCY_INVALID(C)
    A -> DEPENDENCY_INVALID(B)

### Meaning

The dependent assumption has lost one of the assumptions required for its own justification.

### Immediate cause

The current EASL output records the immediate invalid dependency in:

    Invalidation.DependencyID

This preserves local causality.

A full transitive explanation can be reconstructed by following invalidation edges.

---

## 9. Multiple invalidation causes may coexist

An assumption can fail for more than one reason in the same snapshot.

Example:

    assumption A:
      requires: [health]
      valid_until: 12:00:05

    snapshot:
      at: 12:00:06
      health missing

A may receive both:

    ASSUMPTION_EXPIRED
    MISSING_EVIDENCE

This is intentional.

EASL should not erase valid causal information merely to force a single reason.

### Rule

> **Invalidation is a causal set, not necessarily a single primary label.**

Consumers that need a single operational outcome should apply their own policy over the returned causes.

---

## 10. No universal cause precedence

EASL should not define a universal execution precedence such as:

    contradiction > expiry > stale > missing

for action policy.

That would turn epistemic evaluation into operational policy.

However, EASL may define aggregate epistemic summaries for convenience.

The current evaluator uses:

    CONTRADICTORY

when active contradiction exists.

Otherwise, invalidated assumptions produce:

    INSUFFICIENT

This aggregate summary does not erase the full invalidation set.

Consumers should inspect:

    Evaluation.Invalidations

when cause-specific behavior matters.

---

## 11. Aggregate state vs cause-specific invalidations

Current aggregate states are:

    VALID
    DEGRADED
    INVALID

Current evidence summaries are:

    SUFFICIENT
    INSUFFICIENT
    CONTRADICTORY

These are summaries.

They are not substitutes for causal invalidation records.

For example:

    State = INVALID
    EvidenceStatus = INSUFFICIENT

may contain:

    ASSUMPTION_EXPIRED
    DEPENDENCY_INVALID
    MISSING_EVIDENCE

The detailed causes remain authoritative for explanation.

---

## 12. Invalidity propagation is directional

Dependency invalidation follows declared dependency direction.

If:

    A depends on B

then:

    Invalid(B) => Invalid(A)

but not automatically:

    Invalid(A) => Invalid(B)

Dependency edges represent justification requirements, not equivalence.

This directional rule prevents accidental bidirectional invalidation.

---

## 13. Propagation must terminate

The assumption dependency graph must be acyclic.

If the graph contains a cycle:

    A depends on B
    B depends on A

EASL cannot safely determine invalidation propagation.

The evaluator therefore returns an error.

Cycles are structural contract violations, not runtime invalidation causes.

This distinction should remain explicit.

---

## 14. Unknown dependencies are structural errors

If an assumption depends on an assumption ID that is absent from the snapshot, EASL returns an error.

Example:

    A depends on B

but B is not declared.

This is different from missing evidence.

Why?

Because:

    missing evidence

is a normal epistemic condition that can occur in a valid model.

But:

    missing assumption dependency

means the graph itself is malformed or incomplete.

---

## 15. Structural errors are not invalidation reasons

The following are currently evaluation errors, not invalidation reasons:

- zero Snapshot.At;
- empty evidence ID;
- duplicate evidence ID;
- empty assumption ID;
- duplicate assumption ID;
- unknown assumption dependency;
- dependency cycle;
- contradiction targeting an unknown assumption.

These conditions prevent reliable evaluation.

EASL should not pretend that a malformed epistemic model merely produced another ordinary invalid assumption.

---

## 16. Invalidation records should be machine-readable

Current shape:

    Invalidation {
        AssumptionID
        Reason
        EvidenceID
        DependencyID
    }

The fields are intentionally explicit.

Examples:

### Missing evidence

    {
      "assumption_id": "safe-to-execute",
      "reason": "MISSING_EVIDENCE",
      "evidence_id": "cluster-health"
    }

### Stale evidence

    {
      "assumption_id": "safe-to-execute",
      "reason": "STALE_EVIDENCE",
      "evidence_id": "cluster-health"
    }

### Contradiction

    {
      "assumption_id": "safe-to-execute",
      "reason": "CONTRADICTED",
      "evidence_id": "regression-signal"
    }

### Dependency invalidation

    {
      "assumption_id": "safe-to-execute",
      "reason": "DEPENDENCY_INVALID",
      "dependency_id": "routing-stable"
    }

### Assumption expiry

    {
      "assumption_id": "safe-to-execute",
      "reason": "ASSUMPTION_EXPIRED"
    }

---

## 17. Causality should remain explainable

A consumer should be able to build an explanation chain.

Example:

    deploy-safe
      -> DEPENDENCY_INVALID(network-stable)

    network-stable
      -> DEPENDENCY_INVALID(route-current)

    route-current
      -> ASSUMPTION_EXPIRED

Human explanation:

> deploy-safe is invalid because network-stable is invalid, which is invalid because route-current expired.

This explainability is a major reason to keep explicit invalidation causes instead of reducing everything to a confidence score.

---

## 18. Revalidation does not delete prior invalidation

A later snapshot may produce a valid result after new evidence or a new temporal boundary is supplied.

Example:

    snapshot t1:
      A -> ASSUMPTION_EXPIRED

    fresh evaluation performed

    snapshot t2:
      A -> valid

The new result does not mean the old invalidation was wrong.

It means a new epistemic context supports the assumption again.

The current EASL core does not persist historical evaluations.

If history is added later, previous invalidations should remain auditable rather than being silently overwritten.

---

## 19. Invalidation and degradation are different

Not every problem invalidates an assumption.

Current EASL may mark a snapshot:

    DEGRADED

when stale evidence exists but no declared assumption requires it.

This means:

- the snapshot contains epistemically degraded material;
- no assumption currently depends on that stale evidence.

Therefore:

    stale evidence
        does not always imply
    invalid assumption

The distinction prevents unrelated stale evidence from automatically poisoning every assumption.

A consumer may still decide that any degraded state deserves escalation.

That remains consumer policy.

---

## 20. Invalidation and authorization are separate

EASL does not issue permissions.

It produces epistemic state.

A downstream system may map invalidation causes to operational policy.

Example mapping in a consumer:

    CONTRADICTED
        -> BLOCK

    ASSUMPTION_EXPIRED
        -> BLOCK

    MISSING_EVIDENCE
        -> ESCALATE

    DEPENDENCY_INVALID
        -> BLOCK

Another consumer may choose differently.

Therefore the specification must preserve this separation:

    EASL:
      what is invalid and why?

    Consumer:
      what action is permitted because of that?

---

## 21. Current EASL invalidation algorithm

At a high level, current evaluation proceeds as follows:

    1. validate snapshot structure
    2. index evidence
    3. determine stale evidence
    4. index assumptions
    5. validate dependency graph
    6. apply fresh contradiction edges
    7. apply assumption expiry
    8. apply required-evidence checks
    9. propagate dependency invalidity
   10. compute aggregate state

The order exists for implementation clarity.

It must not be interpreted as operational cause precedence.

Multiple causes may remain visible.

---

## 22. Determinism requirements

Given the same snapshot:

- Snapshot.At;
- evidence set;
- evidence expiry boundaries;
- contradiction edges;
- assumption set;
- assumption validity boundaries;
- dependency graph;

the invalidation result should be reproducible.

This enables:

- replay;
- incident reconstruction;
- conformance tests;
- cross-implementation comparison;
- stable downstream policy.

No invalidation rule should depend on hidden wall-clock time.

---

## 23. Ordering of invalidation output

The semantic meaning of invalidation records should not depend on slice order.

Consumers must not interpret:

    Invalidations[0]

as the globally highest-priority cause unless a future specification explicitly defines such ordering.

The current implementation guarantees sorting for some aggregate ID lists, but invalidation records themselves should be treated as a set of causal facts.

If deterministic invalidation ordering later becomes necessary for serialization or conformance, it should be specified explicitly and tested.

---

## 24. Duplicate causal records

The implementation should avoid meaningless duplicate invalidation records when the same exact cause is discovered repeatedly.

However, distinct causes affecting the same assumption must remain distinct.

For example:

    A -> ASSUMPTION_EXPIRED
    A -> MISSING_EVIDENCE(health)

are not duplicates.

A future normalization primitive may define canonical deduplication if real consumers require byte-stable output.

It is not yet required.

---

## 25. Invalidation is snapshot-relative

An invalidation is always relative to an evaluation context.

The correct interpretation is:

> A is invalid in snapshot S at time t for cause C.

Not:

> A is permanently invalid.

A later snapshot can produce a new valid justification.

This keeps invalidation compatible with revalidation.

---

## 26. Failure monotonicity inside a fixed snapshot

Within one immutable snapshot, invalidation should be monotonic.

Once EASL establishes that an assumption is invalid for that snapshot, later dependency propagation within the same evaluation must not make it valid again.

Conceptually:

    invalid(A, S) => remains invalid(A, S)

A new snapshot is required to establish a different state.

---

## 27. Dependency invalidation should preserve local provenance

When propagation invalidates a dependent assumption, the invalidation should identify the immediate upstream dependency that caused the propagation.

Example:

    A depends on B
    B depends on C
    C expired

Preferred causal records:

    C -> ASSUMPTION_EXPIRED
    B -> DEPENDENCY_INVALID(C)
    A -> DEPENDENCY_INVALID(B)

Not:

    A -> DEPENDENCY_INVALID(C)

unless the implementation explicitly computes transitive provenance as an additional field.

Immediate provenance keeps the graph explanation faithful to declared edges.

---

## 28. Evidence contradiction should remain explicit

EASL does not infer semantic contradiction from arbitrary values.

It does not decide that:

    value = unhealthy

contradicts:

    target-is-healthy

unless the producer declares that relationship.

Current model:

    Evidence.Contradicts = [AssumptionID]

This keeps domain interpretation outside EASL.

The producer owns semantic interpretation.

EASL owns deterministic propagation of the declared contradiction.

---

## 29. No implicit invalidation from unrelated evidence

Evidence that is present but not:

- required by an assumption;
- declared as contradicting an assumption;

should not invalidate that assumption merely because the evidence exists.

This avoids hidden semantic coupling.

EASL remains declarative:

    Requires
    DependsOn
    Contradicts
    ValidUntil
    ExpiresAt

are the explicit edges and boundaries that drive evaluation.

---

## 30. Candidate future invalidation reasons

Future consumers may justify additional reasons such as:

    SOURCE_REVOKED
    IDENTITY_CHANGED
    POLICY_INPUT_REPLACED
    VERSION_SUPERSEDED
    TRUST_DOMAIN_LOST

These should not be added merely because they sound useful.

A new reason belongs in EASL only when:

1. the condition is domain-neutral enough to reuse;
2. at least one real consumer needs deterministic semantics for it;
3. it cannot already be represented faithfully by existing reasons;
4. its propagation behavior can be specified clearly;
5. conformance-like tests can define its meaning.

---

## 31. Relationship to tracker-style invalidation

Aegis-EGE currently contains richer runtime invalidation logic tied to observed resource change.

Some of that logic may eventually become generic EASL capability.

The extraction rule is:

> **Move the generic invalidation primitive, not the domain-specific resource model.**

For example, a generic concept such as:

    upstream dependency changed
        -> invalidate dependent assumption

may belong in EASL.

A Kubernetes-specific structure such as:

    APIVersion / Kind / Namespace / UID / ResourceVersion

does not automatically belong in EASL.

The domain adapter should translate domain events into domain-neutral invalidation input.

---

## 32. Conformance properties

Any future EASL implementation in another language should satisfy at least these invalidation properties:

- missing required evidence invalidates with MISSING_EVIDENCE;
- stale required evidence invalidates with STALE_EVIDENCE;
- fresh explicit contradiction invalidates with CONTRADICTED;
- exact ValidUntil boundary invalidates with ASSUMPTION_EXPIRED;
- upstream invalidity propagates with DEPENDENCY_INVALID;
- dependency propagation follows edge direction;
- malformed graphs fail evaluation rather than silently passing;
- multiple valid causes may coexist;
- output cause order is not semantically significant;
- no invalidation rule reads hidden wall-clock time.

These properties are stronger than implementation details and can later seed independent conformance tests.

---

## 33. What is deliberately not specified yet

This document does not yet define:

- invalidation event persistence;
- causal graph serialization;
- transitive explanation compression;
- canonical invalidation ordering;
- distributed invalidation propagation;
- revocation channels;
- source reputation;
- probabilistic invalidation;
- automatic evidence reacquisition;
- execution responses to each cause.

Those belong only when consumer evidence demands them.

---

## 34. Governing rules

The invalidation model can be summarized in four rules:

> **1. Invalidity means lost justification, not necessarily falsehood.**

> **2. Every known invalidation should preserve its cause.**

> **3. Dependency invalidity propagates only along declared dependency edges.**

> **4. EASL explains invalidity; consumers decide operational consequence.**

Or, more compactly:

> **Invalidation must be causal, explicit, directional, and policy-neutral.**
