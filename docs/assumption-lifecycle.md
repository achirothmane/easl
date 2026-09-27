# Assumption Lifecycle

> Status: draft specification companion to the [Assumption Decay Doctrine](assumption-decay-doctrine.md).

This document defines the lifecycle model for assumptions in EASL.

It describes how an assumption moves from being declared, to being justified, to becoming invalid or indeterminate, and how a later re-evaluation creates a new justification.

It does **not** define execution policy.

---

## 1. Why a lifecycle is needed

An assumption is not a permanent boolean.

A statement such as:

    target-is-healthy = true

is incomplete unless the system can also answer:

- when was this justified?
- by which evidence?
- until when is the justification valid?
- which other assumptions does it depend on?
- what invalidated it?
- can it be replayed deterministically?
- has it since been revalidated?

The lifecycle gives EASL a vocabulary for answering those questions without turning EASL into an execution-policy engine.

---

## 2. Conceptual lifecycle

The conceptual lifecycle is:

    DECLARED
        ↓
    EVALUATED
        ↓
    JUSTIFIED
        ↓
      VALID
        ↓
    ┌───────────────┬────────────────┬──────────────────┐
    │               │                │                  │
 EXPIRED       CONTRADICTED      UNSUPPORTED      DEPENDENCY_INVALID
    │               │                │                  │
    └───────────────┴────────────────┴──────────────────┘
                         ↓
                      INVALID
                         ↓
                    RE-EVALUATED
                         ↓
                 NEW JUSTIFICATION

This is a **reasoning model**, not yet a persisted runtime state machine.

The current EASL implementation computes an epistemic evaluation from a snapshot. It does not persist lifecycle transitions internally.

---

## 3. Declared

An assumption is **declared** when the system names a claim and specifies the conditions required for it to remain justifiable.

In the current EASL model:

    Assumption {
        ID
        Requires
        DependsOn
        ValidUntil
    }

Declaration does not imply validity.

For example:

    assumption: safe-to-execute
    requires: [cluster-health]
    depends_on: [routing-stable]
    valid_until: 12:00:05

is only a declaration of what must remain true.

The assumption becomes justified only after evaluation against an actual snapshot.

---

## 4. Evaluated

An assumption is **evaluated** when EASL processes it inside a snapshot containing:

- an explicit evaluation time;
- current evidence;
- the assumption graph;
- temporal validity boundaries.

The evaluation time is supplied through:

    Snapshot.At

This makes lifecycle interpretation deterministic.

The same declaration may be valid at one snapshot time and invalid at another.

---

## 5. Justified

An assumption is **justified** when all of its declared validity conditions hold at the evaluation time.

At minimum, this means:

- all required evidence exists;
- required evidence is fresh;
- no fresh evidence explicitly contradicts the assumption;
- the assumption has not reached its validity boundary;
- all declared upstream assumptions remain valid;
- the assumption graph is structurally evaluable.

Conceptually:

    Justified(A, t) =
        RequiredEvidencePresent(A, t)
        AND RequiredEvidenceFresh(A, t)
        AND NOT Contradicted(A, t)
        AND TemporallyValid(A, t)
        AND DependenciesValid(A, t)
        AND GraphEvaluable(t)

The implementation may evolve, but these conditions define the intended semantics.

---

## 6. Valid

A justified assumption is **valid for the current snapshot**.

This is deliberately snapshot-relative.

The correct statement is:

> "This assumption is valid at this evaluation time under this evidence and dependency state."

The incorrect statement is:

> "This assumption is true permanently."

In current EASL output, a snapshot with no invalidated assumptions and no aggregate degradation is represented as:

    State = VALID

This aggregate state does not mean the system has granted any execution authority.

---

## 7. Expired

An assumption becomes **expired** when its explicit validity boundary is reached.

Current rule:

    expired(A, t) iff t >= A.ValidUntil

Therefore:

    now < valid_until   -> not expired
    now = valid_until   -> expired
    now > valid_until   -> expired

The exact boundary is intentionally deterministic.

Current invalidation reason:

    ASSUMPTION_EXPIRED

Expiry means the previous justification is no longer current enough.

It does not necessarily mean the claim became false.

---

## 8. Contradicted

An assumption becomes **contradicted** when fresh evidence explicitly declares a contradiction edge against it.

Example:

    evidence: regression-signal
    contradicts: [safe-to-deploy]

If the evidence is fresh, the assumption is invalidated.

Current invalidation reason:

    CONTRADICTED

Current aggregate evidence status:

    CONTRADICTORY

Contradiction is distinct from missing evidence.

A missing health check says:

> "We cannot currently justify health."

A contradiction says:

> "We have current evidence against the health assumption."

That distinction must remain visible to consumers.

---

## 9. Unsupported

An assumption becomes **unsupported** when evidence required by its declaration is unavailable or stale.

Two current causes exist:

    MISSING_EVIDENCE
    STALE_EVIDENCE

Unsupported means EASL cannot establish the assumption from its declared support set.

It must not be silently treated as contradiction.

Conceptually:

    unsupported != false

Instead:

    unsupported = insufficient justification

Downstream systems may decide to gather evidence, escalate, deny action, or use another policy.

That decision remains outside EASL.

---

## 10. Dependency-invalid

An assumption becomes **dependency-invalid** when one of its declared upstream assumptions is invalid.

Example:

    A depends on B
    B depends on C

If C expires:

    C -> ASSUMPTION_EXPIRED
    B -> DEPENDENCY_INVALID
    A -> DEPENDENCY_INVALID

Current invalidation reason:

    DEPENDENCY_INVALID

The lifecycle must preserve the causal relationship.

A downstream consumer should eventually be able to reconstruct:

    A invalid
      because B invalid
        because C expired

The current EASL output exposes the immediate invalidated dependency through DependencyID.

Deeper causal provenance may be extended later if real consumers require it.

---

## 11. Invalid

**Invalid** is the resulting lifecycle condition when an assumption can no longer be justified under the current snapshot.

Multiple causes can produce invalidity:

- expiry;
- contradiction;
- missing evidence;
- stale required evidence;
- invalid dependency.

In current EASL output, an invalid assumption appears in:

    InvalidatedAssumptions

with machine-readable causes in:

    Invalidations

and the aggregate snapshot state becomes:

    INVALID

when one or more assumptions are invalid.

---

## 12. Structurally indeterminate

Not every failure should be represented as an invalid assumption.

Some inputs prevent EASL from establishing any reliable evaluation.

Examples include:

- duplicate evidence identifiers;
- duplicate assumption identifiers;
- unknown assumption dependencies;
- dependency cycles;
- contradiction edges targeting unknown assumptions;
- zero evaluation time.

These are structural contract errors.

The lifecycle interpretation is:

    declared
       ↓
    evaluation attempted
       ↓
    structurally indeterminate

EASL returns an error instead of pretending the snapshot is valid or invalid.

A downstream consumer such as Aegis-EGE can then fail closed or escalate according to its own policy.

---

## 13. Degraded snapshot vs invalid assumption

Current EASL also has an aggregate state:

    DEGRADED

This currently occurs when stale evidence exists but does not invalidate a declared assumption.

Example:

    stale evidence exists
    but no assumption requires it

The snapshot is not fully clean, but the stale evidence is not currently part of the justification basis for an assumption.

Therefore:

    DEGRADED != INVALID

This distinction should remain explicit.

A consumer may still choose to treat degraded state conservatively.

---

## 14. Re-evaluation

An invalid or expired assumption is not repaired by changing its old evaluation retroactively.

Instead, the system performs a **new evaluation**.

Conceptually:

    old snapshot:
        A = invalid

    new evidence arrives

    new snapshot:
        A = valid

This is re-evaluation.

The new result supersedes the old result for current decision-making, but it does not erase historical fact.

---

## 15. Revalidation

**Revalidation** is the successful outcome of re-evaluation after a previous justification became unusable.

Example:

    t0: A valid
    t1: A expires
    t2: fresh evidence collected
    t3: A evaluated again and valid

The lifecycle is:

    VALID
      ↓
    EXPIRED
      ↓
    RE-EVALUATED
      ↓
    VALID under new justification

A later implementation may attach explicit evaluation identities or versions.

The current core does not yet persist these transitions.

---

## 16. No resurrection by time reversal

An expired assumption should not become valid merely because a caller chooses an earlier wall-clock time during normal operation.

Replay is different from live evaluation.

Because Snapshot.At is explicit, a historical snapshot may be replayed at its original evaluation time.

That replay can validly show:

    A was valid at t0

while a current evaluation shows:

    A is expired at t1

These are not contradictory statements.

They refer to different evaluation times.

---

## 17. Lifecycle and evidence lifetime are separate

Evidence has its own freshness lifecycle.

An evidence item may transition conceptually through:

    observed
       ↓
    fresh
       ↓
    stale

An assumption has a separate lifecycle:

    declared
       ↓
    justified
       ↓
    valid
       ↓
    invalid

These lifecycles interact but must not be collapsed.

For example:

    evidence still fresh
    assumption expired

is valid when the assumption has a stricter temporal boundary.

Likewise:

    assumption nominal window still open
    required evidence stale

must invalidate the assumption.

---

## 18. Lifecycle and consumer policy are separate

EASL may determine:

    ASSUMPTION_EXPIRED

A consumer may map that to:

    BLOCK

Another consumer may map the same epistemic state to:

    REVALIDATE

Another may map it to:

    ESCALATE

The lifecycle does not prescribe those actions.

The layering remains:

    EASL lifecycle
          ↓
    consumer policy
          ↓
    operational decision

For Aegis-EGE today:

    epistemic state
          ↓
    execution policy
          ↓
    ALLOW / BLOCK / ESCALATE

---

## 19. Current lifecycle mapping to EASL v0

The current implementation can be mapped as follows:

| Lifecycle concept | Current EASL representation |
| --- | --- |
| Declared | Assumption |
| Evaluated | Snapshot + Evaluate |
| Valid | StateValid with no invalidation for the assumption |
| Expired | ReasonAssumptionExpired |
| Contradicted | ReasonContradicted |
| Unsupported: missing | ReasonMissingEvidence |
| Unsupported: stale | ReasonStaleEvidence |
| Dependency-invalid | ReasonDependencyInvalid |
| Invalid | InvalidatedAssumptions + Invalidations |
| Structurally indeterminate | Evaluate returns error |
| Degraded snapshot | StateDegraded |
| Revalidated | new successful Evaluate result; not persisted as a lifecycle event |

This table describes the current implementation, not a commitment that every conceptual state must become a stored enum.

---

## 20. Transition invariants

The following lifecycle invariants should hold.

### 20.1 Declaration does not imply justification

Creating an Assumption object must not automatically make it valid.

### 20.2 Evaluation is time-relative

The same assumption may produce different states at different Snapshot.At values.

### 20.3 Invalidity is causal

Every invalidated assumption should have at least one machine-readable invalidation cause.

### 20.4 Expiry is monotonic within one timeline

For a fixed ValidUntil, once:

    Snapshot.At >= ValidUntil

later snapshot times must also treat the assumption as expired unless a new assumption justification is created.

### 20.5 Dependency invalidation propagates downstream

An invalid upstream assumption cannot be silently ignored by a declared dependent assumption.

### 20.6 Revalidation creates a new evaluation

A fresh valid result does not rewrite the previous invalid result.

### 20.7 Structural errors do not become fake lifecycle states

Malformed input should remain an evaluation error unless there is a concrete reason to model it differently.

---

## 21. What is not yet implemented

This lifecycle document does not imply that EASL currently persists:

- lifecycle event streams;
- assumption generation numbers;
- evaluation IDs;
- transition timestamps beyond the input boundaries already supplied;
- historical snapshots;
- revalidation lineage;
- previous-state references;
- transition subscriptions.

These should only be implemented when a real consumer requires them.

The specification may describe the model before the implementation stores every transition.

---

## 22. Candidate future primitives

If consumer evidence eventually justifies them, lifecycle-related primitives may include:

    EvaluationID
    AssumptionGeneration
    RevalidatedFrom
    InvalidatedAt
    JustifiedAt
    lifecycle event records
    causal invalidation paths

None of these are required merely to make the lifecycle document complete.

They should be added only when they reduce ambiguity or enable a real operational need.

---

## 23. Relationship to other EASL documents

This document answers:

> **What stages can an assumption pass through?**

The companion documents answer different questions:

- Assumption Decay Doctrine:
  why justification decays and what principles govern it.
- Invalidation Semantics:
  what each invalidation cause means and how causes compose.
- Temporal Validity:
  how time boundaries and freshness are evaluated.

The documents should remain separable so that lifecycle, invalidation, and temporal semantics can evolve without becoming one monolithic specification.

---

## 24. Governing lifecycle rule

The lifecycle can be reduced to one operational statement:

> **An assumption is valid only for a specific evaluation context; when any declared justification condition fails, the old justification ends and a new evaluation is required to establish validity again.**

Or, compactly:

> **Validity is evaluated, lost causally, and re-earned through revalidation.**
