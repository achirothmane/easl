# Temporal Validity

> Status: draft specification companion to the Assumption Decay Doctrine, Assumption Lifecycle, and Invalidation Semantics.

This document defines the temporal semantics used by EASL.

It specifies how evaluation time, evidence freshness, and assumption lifetime interact. It also defines exact boundary behavior, replay semantics, and the separation between domain policy and epistemic time evaluation.

It does **not** define action sensitivity, execution policy, scheduling, or automatic refresh behavior.

---

## 1. Purpose

Many operational facts are only useful for a limited period.

Examples:

- a node was healthy 3 seconds ago;
- a dependency version matched 30 seconds ago;
- a policy snapshot was current at deployment time;
- a safety assumption was justified before a resource changed;
- an external observation is still fresh enough to support a decision.

Without explicit temporal semantics, systems tend to hide time inside ad hoc age checks. That makes behavior harder to replay, compare, test, and reason about.

EASL instead treats time as part of the epistemic input.

> **Temporal validity must be derived from explicit time boundaries evaluated against an explicit snapshot time.**

---

## 2. The three temporal coordinates

Current EASL has three important temporal fields:

    Snapshot.At
    Evidence.ExpiresAt
    Assumption.ValidUntil

They have different roles.

### Snapshot.At

Snapshot.At is the time at which the caller asks:

> What is justified at this instant?

It is the evaluation reference point for all temporal checks.

EASL does not read the wall clock internally.

### Evidence.ExpiresAt

Evidence.ExpiresAt defines the last instant before which an evidence item remains fresh.

It is a property of evidence freshness.

### Assumption.ValidUntil

Assumption.ValidUntil defines the last instant before which an assumption's current justification remains temporally valid.

It is a property of assumption lifetime.

These fields must not be collapsed into one concept.

---

## 3. Explicit evaluation time

Every EASL evaluation requires:

    Snapshot.At != zero

A zero evaluation time is a structural error.

Without an explicit evaluation instant, EASL cannot deterministically answer whether evidence is stale, whether an assumption has expired, or whether a replay should produce the same result.

Therefore:

> **Temporal evaluation without an explicit reference time is invalid input.**

---

## 4. Evidence freshness

Evidence may optionally declare ExpiresAt.

If no expiry is declared, the current EASL core does not infer one.

If an expiry exists, evidence is stale when:

    Snapshot.At >= Evidence.ExpiresAt

Therefore:

    t < ExpiresAt  -> fresh
    t = ExpiresAt  -> stale
    t > ExpiresAt  -> stale

The exact boundary is intentionally closed on expiry.

---

## 5. Assumption expiry

An assumption may optionally declare ValidUntil.

If no validity boundary is declared, the current EASL core does not invent one.

If a boundary exists, the assumption is expired when:

    Snapshot.At >= Assumption.ValidUntil

Therefore:

    t < ValidUntil  -> not expired
    t = ValidUntil  -> expired
    t > ValidUntil  -> expired

This exact-boundary rule is already enforced by the implementation and tests.

---

## 6. Why both boundaries exist

Evidence freshness and assumption lifetime answer different questions.

Evidence asks:

> Is this observation still fresh?

Assumption lifetime asks:

> Is this justification still allowed to remain valid?

A system may intentionally choose:

    Evidence.ExpiresAt = 12:01:00
    Assumption.ValidUntil = 12:00:10

At 12:00:20 the evidence is still fresh, but the assumption is already expired.

That can be correct. A health signal may remain useful for one minute while a critical destructive operation only trusts a derived safety assumption for ten seconds.

Therefore freshness and assumption lifetime must remain separate concepts.

---

## 7. Domain policy chooses duration

EASL should not decide how long a domain-specific assumption deserves to live.

A consumer may define policy such as:

    LOW      -> 60 seconds
    MEDIUM   -> 30 seconds
    HIGH     -> 10 seconds
    CRITICAL -> 5 seconds

That mapping belongs to the consumer.

The consumer can derive:

    ValidUntil = EvaluatedAt + MaxAge

and supply the absolute boundary to EASL.

The layering is:

    domain policy
        ↓
    choose lifetime
        ↓
    derive absolute ValidUntil
        ↓
    EASL evaluates boundary
        ↓
    current epistemic state

This keeps EASL policy-neutral.

---

## 8. Absolute boundaries are preferred

An absolute boundary such as:

    valid_until = 2026-09-27T03:20:05Z

is preferable inside EASL to a hidden relative rule such as:

    max_age = 5 seconds

because an absolute boundary makes the snapshot self-describing.

A replay does not need to know which risk policy was active when the lifetime was chosen. It only needs the evaluation time and the absolute validity boundary.

---

## 9. Replay semantics

Replay is one of the main reasons EASL uses caller-supplied time.

Suppose:

    EvaluatedAt = 12:00:00
    ValidUntil  = 12:00:05

A historical replay at 12:00:03 should show the assumption as temporally valid.

A current evaluation at 12:01:00 should show it as expired.

Both results are correct because they answer different questions.

Replay means:

> Evaluate the same epistemic model at an explicitly supplied historical instant.

It does not mean mutating the current clock or pretending the current state never changed.

---

## 10. Historical replay is not live resurrection

The explicit-time model permits historical replay.

That does not mean a live system should reuse an expired assumption merely by supplying an older timestamp.

The distinction is contextual:

    historical replay
        -> asks what was justified then

    live decision
        -> asks what is justified now

Consumers are responsible for supplying the correct evaluation time for their purpose.

---

## 11. UTC and normalization

The current Go implementation compares time.Time instants directly.

Consumers should normalize externally sourced timestamps consistently, preferably to UTC, before constructing snapshots.

The semantic rule is about instants, not display time zones.

For example:

    2026-09-27T12:00:00+01:00

and:

    2026-09-27T11:00:00Z

represent the same instant.

Implementations must compare instants, not formatted strings.

---

## 12. Clock source responsibility

EASL does not own a clock.

The caller owns the source of Snapshot.At.

Different consumers may use:

- a real-time system clock;
- an incident replay timestamp;
- a deterministic test fixture;
- a simulation clock.

EASL's responsibility is only:

> Given this explicit evaluation instant, evaluate temporal validity deterministically.

---

## 13. Clock skew

The current EASL core does not independently reject every possible clock-skew scenario.

For example, it does not currently enforce:

    Evidence.ObservedAt <= Snapshot.At

This is an intentional boundary.

A consumer may detect clock skew before building the EASL snapshot.

A future EASL primitive may validate observed-time relationships if multiple real consumers require identical semantics.

Until then, hidden clock-skew policy should not be added speculatively.

---

## 14. ObservedAt vs ExpiresAt

Current evidence contains:

    ObservedAt
    ExpiresAt

But freshness is currently determined by the explicit expiry boundary, not by deriving age internally from ObservedAt.

Conceptually:

    ObservedAt
        -> when the evidence was produced

    ExpiresAt
        -> until when the evidence may remain fresh

The producer or consumer may derive:

    ExpiresAt = ObservedAt + freshness_window

before constructing the snapshot.

This keeps freshness policy outside EASL.

---

## 15. Missing expiry boundaries

A missing expiry boundary does not mean permanent truth.

It means:

> no temporal expiry rule was declared for this object in the current model.

That may be valid for assumptions whose invalidation is event-driven or dependency-driven.

Consumers should not interpret the absence of ExpiresAt or ValidUntil as proof that time can never make a claim obsolete.

It simply means EASL has no temporal boundary to apply.

---

## 16. Time and contradiction

A contradiction edge is active only while the contradicting evidence is fresh.

Current behavior:

    evidence stale
        ->
    its contradiction edges are ignored

Conceptually:

    Fresh(E)
    AND E explicitly contradicts A
        ->
    A is contradicted

Old contradictory evidence does not remain permanently authoritative.

If a domain requires permanent revocation semantics, that should be represented by a different primitive rather than by pretending stale evidence is still fresh.

---

## 17. Time and dependency invalidation

Dependency invalidation is not inherently time-based.

However, time can trigger it indirectly.

Example:

    A depends on B
    B.ValidUntil = 12:00:05

At 12:00:05:

    B -> ASSUMPTION_EXPIRED
    A -> DEPENDENCY_INVALID(B)

Thus temporal invalidation can propagate structurally through the assumption graph.

---

## 18. Time and missing evidence

Missing evidence is not temporal by itself.

But time can cause evidence to become unusable without disappearing.

Two states must remain distinct:

    evidence absent
        -> MISSING_EVIDENCE

    evidence present but expired
        -> STALE_EVIDENCE

Both may invalidate an assumption, but they carry different operational meaning.

---

## 19. Exact boundary semantics

EASL uses the same general boundary rule for evidence and assumptions:

    fresh/valid strictly before the boundary
    stale/expired at or after the boundary

In pseudocode:

    valid := now < boundary

not:

    valid := now <= boundary

This avoids ambiguous one-instant grace behavior and makes conformance tests straightforward.

---

## 20. No hidden grace period

Grace periods are policy.

If a consumer wants five seconds plus one second of tolerance, it should derive:

    ValidUntil = base_time + 6 seconds

EASL should not secretly add a grace period during evaluation.

Hidden grace rules would reduce replayability and make cross-implementation behavior inconsistent.

---

## 21. No automatic refresh

EASL does not automatically reacquire evidence when it becomes stale.

It also does not automatically extend ValidUntil.

Those behaviors require domain knowledge and side effects.

The correct layering is:

    EASL detects stale / expired
        ↓
    consumer decides whether revalidation is needed
        ↓
    evidence producer gathers new evidence
        ↓
    a new snapshot is evaluated

---

## 22. Revalidation and time

Successful revalidation should create a new temporal basis.

Example:

    old:
      evaluated_at = 12:00:00
      valid_until  = 12:00:05

    revalidated at 12:00:07

    new:
      evaluated_at = 12:00:07
      valid_until  = 12:00:12

The new validity window does not retroactively change the old one.

Historical replay should still show the old assumption as expired after 12:00:05.

---

## 23. Temporal monotonicity

For a fixed validity boundary T, once:

    Snapshot.At >= T

all later evaluation instants on the same forward-moving timeline must also treat that assumption as expired.

Likewise for evidence freshness.

A new valid state requires a new epistemic object or new snapshot with a new boundary.

---

## 24. Snapshot immutability

For one call to Evaluate:

- Snapshot.At is fixed;
- evidence boundaries are fixed;
- assumption boundaries are fixed;
- dependency declarations are fixed.

EASL should evaluate that immutable input to one deterministic output.

Changing time or boundaries means constructing a new snapshot.

---

## 25. Temporal state is not a scheduler

EASL can determine:

    this assumption is expired now

It does not schedule:

    re-evaluate this assumption in 5 seconds

Scheduling belongs to orchestration.

A consumer may use ValidUntil to determine a useful revalidation time, but EASL does not own timers, queues, jobs, or wakeups.

---

## 26. Temporal state is not authorization TTL

Execution systems may also have authorization lifetimes.

Those must not be confused with assumption validity.

Example:

    assumption ValidUntil
        -> how long the epistemic claim remains justified

    authorization ExpiresAt
        -> how long an issued execution permit may be used

These are different boundaries.

A valid assumption does not imply a valid authorization.

---

## 27. Temporal hierarchy

A real system may contain several lifetimes:

    evidence freshness
        ↓
    assumption lifetime
        ↓
    decision lifetime
        ↓
    authorization lifetime
        ↓
    execution lock lifetime

These should remain separate unless a domain explicitly chooses to align them.

EASL currently owns only the first two epistemic layers.

---

## 28. Temporal validity and aggregate state

Time can affect aggregate EASL state in different ways.

### Stale required evidence

    evidence stale
    AND assumption requires it
        ->
    assumption invalid
        ->
    State = INVALID

### Stale unrelated evidence

    evidence stale
    AND no assumption requires it
        ->
    State = DEGRADED

### Expired assumption

    assumption expired
        ->
    assumption invalid
        ->
    State = INVALID

This distinction preserves whether temporal degradation actually invalidates a declared justification.

---

## 29. Determinism requirement

Given identical:

- Snapshot.At;
- Evidence.ExpiresAt;
- Assumption.ValidUntil;
- evidence set;
- assumption graph;
- contradiction edges;

the temporal result must be identical.

No EASL implementation should read hidden wall-clock time inside the evaluator.

Hidden clock reads would break deterministic replay.

---

## 30. Serialization

When snapshots cross process or language boundaries, timestamps should use an unambiguous instant representation.

Recommended form:

    RFC 3339 / ISO 8601 with explicit offset

Examples:

    2026-09-27T03:20:05Z
    2026-09-27T04:20:05+01:00

Consumers should avoid ambiguous local timestamps without an offset or timezone.

A future wire specification may make this normative.

---

## 31. Precision

Temporal comparisons should preserve enough precision for the domain.

The current Go implementation uses time.Time and can represent sub-second precision.

The semantic rule remains:

    t >= boundary -> stale / expired

Cross-language implementations must not silently round timestamps in a way that moves them across the boundary.

---

## 32. Distributed systems

Distributed systems may disagree about current time.

EASL does not solve clock synchronization.

It evaluates the time provided by the caller.

Consumers operating across distributed nodes should establish their own clock discipline, such as synchronized clocks, bounded skew assumptions, authoritative timestamps, or logical sequencing where appropriate.

If bounded clock uncertainty later becomes a real multi-consumer need, EASL could gain explicit uncertainty semantics.

It should not assume them now.

---

## 33. Temporal uncertainty

The current EASL model uses exact boundaries.

It does not model approximate validity such as:

    clock_error = plus/minus 500ms

A future model could represent earliest expiry, latest expiry, or clock uncertainty.

Such a model would materially expand EASL semantics and should only be introduced when real systems cannot safely operate under exact timestamp assumptions.

---

## 34. Future temporal primitives

Potential future primitives include:

    ValidFrom
    JustifiedAt
    RevalidatedAt
    ClockUncertainty
    SourceTimestamp
    ObservationWindow
    TemporalGeneration

None should be added for theoretical completeness alone.

A primitive belongs in EASL only when it is required by a real consumer, is domain-neutral, has deterministic semantics, and can be tested for conformance.

---

## 35. Conformance properties

Any future EASL implementation should satisfy at least these temporal properties:

- zero Snapshot.At is rejected;
- evidence is fresh strictly before ExpiresAt;
- evidence is stale exactly at ExpiresAt;
- assumption is valid strictly before ValidUntil;
- assumption is expired exactly at ValidUntil;
- no hidden wall clock participates in evaluation;
- stale contradictory evidence does not create an active contradiction;
- temporal invalidation may propagate through DependsOn;
- evidence expiry and assumption expiry remain separate;
- missing expiry fields do not create an implicit timeout;
- historical replay is deterministic.

These properties can later seed independent conformance tests.

---

## 36. Current implementation mapping

| Temporal concept | Current EASL representation |
| --- | --- |
| Evaluation instant | Snapshot.At |
| Evidence observation time | Evidence.ObservedAt |
| Evidence freshness boundary | Evidence.ExpiresAt |
| Assumption validity boundary | Assumption.ValidUntil |
| Stale required evidence | ReasonStaleEvidence |
| Expired assumption | ReasonAssumptionExpired |
| No expiry declared | nil boundary; no inferred timeout |
| Replay | caller supplies historical Snapshot.At |
| Hidden wall clock | not used by Evaluate |

---

## 37. Relationship to the specification set

The EASL specification set now has four responsibilities.

**Assumption Decay Doctrine** asks why justification decays.

**Assumption Lifecycle** asks through which conceptual states an assumption can move.

**Invalidation Semantics** asks why an assumption is no longer justified.

**Temporal Validity** asks how time determines whether evidence or an assumption remains usable.

These documents should remain separate so each dimension can evolve without becoming one monolithic specification.

---

## 38. Outside this document

This document does not define:

- LOW / MEDIUM / HIGH / CRITICAL policy;
- risk-based TTL selection;
- automatic evidence refresh;
- revalidation scheduling;
- authorization expiry;
- execution lock expiry;
- distributed clock synchronization;
- probabilistic decay;
- human approval timeout;
- domain-specific freshness windows;
- polling cadence.

Those belong to consumers, orchestration, or future primitives justified by evidence.

---

## 39. Governing rules

Temporal validity can be summarized in five rules:

> **1. Evaluation time must be explicit.**

> **2. Freshness and assumption lifetime are separate boundaries.**

> **3. A boundary expires at the exact instant it is reached.**

> **4. Consumers choose lifetimes; EASL evaluates absolute boundaries.**

> **5. Replay uses supplied time, never hidden wall-clock time.**

Or, compactly:

> **Time is part of the evidence contract, not an invisible side effect.**
