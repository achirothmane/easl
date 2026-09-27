package easl

import "time"

// EvidenceID is a stable identifier for one evidence item.
type EvidenceID string

// AssumptionID is a stable identifier for one assumption.
type AssumptionID string

// State is the aggregate epistemic state of a snapshot.
type State string

const (
	StateValid    State = "VALID"
	StateDegraded State = "DEGRADED"
	StateInvalid  State = "INVALID"
)

// EvidenceStatus summarizes whether the snapshot contains enough
// non-contradictory evidence for its assumptions.
type EvidenceStatus string

const (
	EvidenceSufficient    EvidenceStatus = "SUFFICIENT"
	EvidenceInsufficient  EvidenceStatus = "INSUFFICIENT"
	EvidenceContradictory EvidenceStatus = "CONTRADICTORY"
)

// InvalidationReason explains why an assumption cannot currently be trusted.
type InvalidationReason string

const (
	ReasonMissingEvidence   InvalidationReason = "MISSING_EVIDENCE"
	ReasonStaleEvidence     InvalidationReason = "STALE_EVIDENCE"
	ReasonContradicted      InvalidationReason = "CONTRADICTED"
	ReasonDependencyInvalid InvalidationReason = "DEPENDENCY_INVALID"
)

// Evidence is an externally produced observation. EASL does not interpret
// domain payloads; it only tracks identity, freshness, and explicit
// contradiction edges.
type Evidence struct {
	ID EvidenceID `json:"id"`

	ObservedAt time.Time  `json:"observed_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`

	// Contradicts names assumptions this evidence explicitly invalidates while
	// the evidence is fresh. Contradiction semantics are supplied by producers,
	// not inferred by EASL.
	Contradicts []AssumptionID `json:"contradicts,omitempty"`
}

// Assumption declares the evidence and upstream assumptions that must remain
// valid for this assumption to remain valid.
type Assumption struct {
	ID AssumptionID `json:"id"`

	Requires  []EvidenceID   `json:"requires,omitempty"`
	DependsOn []AssumptionID `json:"depends_on,omitempty"`
}

// Snapshot is the deterministic input to Evaluate. At is supplied by the
// caller so replaying the same snapshot at the same instant returns the same
// result.
type Snapshot struct {
	At          time.Time    `json:"at"`
	Evidence    []Evidence   `json:"evidence,omitempty"`
	Assumptions []Assumption `json:"assumptions,omitempty"`
}

// Contradiction is an active contradiction edge at evaluation time.
type Contradiction struct {
	EvidenceID   EvidenceID   `json:"evidence_id"`
	AssumptionID AssumptionID `json:"assumption_id"`
}

// Invalidation records a machine-readable reason an assumption is invalid.
type Invalidation struct {
	AssumptionID AssumptionID       `json:"assumption_id"`
	Reason       InvalidationReason `json:"reason"`
	EvidenceID   EvidenceID         `json:"evidence_id,omitempty"`
	DependencyID AssumptionID       `json:"dependency_id,omitempty"`
}

// Evaluation is the domain-neutral epistemic state consumed by downstream
// policy engines such as Aegis-EGE.
type Evaluation struct {
	State                  State           `json:"state"`
	EvidenceStatus         EvidenceStatus  `json:"evidence_status"`
	StaleEvidence          []EvidenceID    `json:"stale_evidence,omitempty"`
	Contradictions         []Contradiction `json:"contradictions,omitempty"`
	InvalidatedAssumptions []AssumptionID  `json:"invalidated_assumptions,omitempty"`
	Invalidations          []Invalidation  `json:"invalidations,omitempty"`
}
