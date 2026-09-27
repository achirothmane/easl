package easl

import (
	"testing"
	"time"
)

func TestEvaluateValid(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 45, 0, 0, time.UTC)
	expires := now.Add(time.Hour)

	got, err := Evaluate(Snapshot{
		At: now,
		Evidence: []Evidence{{
			ID:         "health-check",
			ObservedAt: now.Add(-time.Minute),
			ExpiresAt:  &expires,
		}},
		Assumptions: []Assumption{{
			ID:       "target-healthy",
			Requires: []EvidenceID{"health-check"},
		}},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.State != StateValid || got.EvidenceStatus != EvidenceSufficient {
		t.Fatalf("unexpected evaluation: %+v", got)
	}
}

func TestEvaluateStaleRequiredEvidenceInvalidates(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 45, 0, 0, time.UTC)
	expires := now.Add(-time.Second)

	got, err := Evaluate(Snapshot{
		At: now,
		Evidence: []Evidence{{
			ID:         "health-check",
			ObservedAt: now.Add(-time.Minute),
			ExpiresAt:  &expires,
		}},
		Assumptions: []Assumption{{
			ID:       "target-healthy",
			Requires: []EvidenceID{"health-check"},
		}},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.State != StateInvalid || got.EvidenceStatus != EvidenceInsufficient {
		t.Fatalf("unexpected evaluation: %+v", got)
	}
}

func TestEvaluateContradictionWins(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 45, 0, 0, time.UTC)

	got, err := Evaluate(Snapshot{
		At: now,
		Evidence: []Evidence{{
			ID:          "regression-signal",
			ObservedAt:  now,
			Contradicts: []AssumptionID{"safe-to-change"},
		}},
		Assumptions: []Assumption{{ID: "safe-to-change"}},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.State != StateInvalid || got.EvidenceStatus != EvidenceContradictory {
		t.Fatalf("unexpected evaluation: %+v", got)
	}
}

func TestEvaluatePropagatesDependencyInvalidation(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 45, 0, 0, time.UTC)

	got, err := Evaluate(Snapshot{
		At: now,
		Evidence: []Evidence{{
			ID:          "bad-signal",
			ObservedAt:  now,
			Contradicts: []AssumptionID{"base"},
		}},
		Assumptions: []Assumption{
			{ID: "base"},
			{ID: "derived", DependsOn: []AssumptionID{"base"}},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(got.InvalidatedAssumptions) != 2 {
		t.Fatalf("expected two invalid assumptions, got %+v", got.InvalidatedAssumptions)
	}
}

func TestEvaluateMissingRequiredEvidenceIsInsufficient(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 45, 0, 0, time.UTC)

	got, err := Evaluate(Snapshot{
		At:          now,
		Assumptions: []Assumption{{ID: "a", Requires: []EvidenceID{"missing"}}},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.State != StateInvalid || got.EvidenceStatus != EvidenceInsufficient {
		t.Fatalf("unexpected evaluation: %+v", got)
	}
	if len(got.Invalidations) != 1 || got.Invalidations[0].Reason != ReasonMissingEvidence {
		t.Fatalf("expected missing-evidence invalidation, got %+v", got.Invalidations)
	}
}

func TestEvaluateRejectsDependencyCycle(t *testing.T) {
	now := time.Date(2026, 9, 27, 1, 45, 0, 0, time.UTC)

	_, err := Evaluate(Snapshot{
		At: now,
		Assumptions: []Assumption{
			{ID: "a", DependsOn: []AssumptionID{"b"}},
			{ID: "b", DependsOn: []AssumptionID{"a"}},
		},
	})
	if err == nil {
		t.Fatal("expected dependency-cycle error")
	}
}

func TestEvaluateAssumptionValidBeforeExpiry(t *testing.T) {
	now := time.Date(2026, 9, 27, 2, 45, 0, 0, time.UTC)
	validUntil := now.Add(time.Second)

	got, err := Evaluate(Snapshot{
		At: now,
		Assumptions: []Assumption{{
			ID:         "safe-to-execute",
			ValidUntil: &validUntil,
		}},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.State != StateValid {
		t.Fatalf("expected %s before assumption expiry, got %+v", StateValid, got)
	}
}

func TestEvaluateAssumptionExpiresAtBoundary(t *testing.T) {
	now := time.Date(2026, 9, 27, 2, 45, 0, 0, time.UTC)
	validUntil := now

	got, err := Evaluate(Snapshot{
		At: now,
		Assumptions: []Assumption{{
			ID:         "safe-to-execute",
			ValidUntil: &validUntil,
		}},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.State != StateInvalid {
		t.Fatalf("expected %s at assumption expiry boundary, got %+v", StateInvalid, got)
	}
	if len(got.Invalidations) != 1 || got.Invalidations[0].Reason != ReasonAssumptionExpired {
		t.Fatalf("expected assumption-expired invalidation, got %+v", got.Invalidations)
	}
}

func TestEvaluateExpiredAssumptionInvalidatesDependents(t *testing.T) {
	now := time.Date(2026, 9, 27, 2, 45, 0, 0, time.UTC)
	expired := now.Add(-time.Second)

	got, err := Evaluate(Snapshot{
		At: now,
		Assumptions: []Assumption{
			{
				ID:         "base",
				ValidUntil: &expired,
			},
			{
				ID:        "derived",
				DependsOn: []AssumptionID{"base"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(got.InvalidatedAssumptions) != 2 {
		t.Fatalf("expected base and dependent assumptions invalidated, got %+v", got.InvalidatedAssumptions)
	}

	reasons := map[AssumptionID]InvalidationReason{}
	for _, invalidation := range got.Invalidations {
		reasons[invalidation.AssumptionID] = invalidation.Reason
	}
	if reasons["base"] != ReasonAssumptionExpired {
		t.Fatalf("expected base expiry, got %q", reasons["base"])
	}
	if reasons["derived"] != ReasonDependencyInvalid {
		t.Fatalf("expected dependency invalidation, got %q", reasons["derived"])
	}
}


func TestEvaluateMatchingSubjectStateBindingRemainsValid(t *testing.T) {
	now := time.Date(2026, 9, 27, 4, 10, 0, 0, time.UTC)

	got, err := Evaluate(Snapshot{
		At: now,
		StateBindings: []StateBinding{{
			ID:       "target-state",
			Expected: "sha256:abc",
			Observed: "sha256:abc",
		}},
		Assumptions: []Assumption{{
			ID:                    "safe-to-act",
			RequiresStateBindings: []StateBindingID{"target-state"},
		}},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.State != StateValid {
		t.Fatalf("expected matching subject-state binding to remain valid, got %+v", got)
	}
}

func TestEvaluateChangedSubjectStateBindingInvalidatesAssumption(t *testing.T) {
	now := time.Date(2026, 9, 27, 4, 10, 0, 0, time.UTC)

	got, err := Evaluate(Snapshot{
		At: now,
		StateBindings: []StateBinding{{
			ID:       "target-state",
			Expected: "sha256:before",
			Observed: "sha256:after",
		}},
		Assumptions: []Assumption{{
			ID:                    "safe-to-act",
			RequiresStateBindings: []StateBindingID{"target-state"},
		}},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got.State != StateInvalid || got.EvidenceStatus != EvidenceInsufficient {
		t.Fatalf("expected changed subject state to invalidate justification, got %+v", got)
	}
	if len(got.Invalidations) != 1 {
		t.Fatalf("expected one invalidation, got %+v", got.Invalidations)
	}
	invalidation := got.Invalidations[0]
	if invalidation.AssumptionID != "safe-to-act" ||
		invalidation.Reason != ReasonSubjectStateChanged ||
		invalidation.StateBindingID != "target-state" {
		t.Fatalf("unexpected subject-state invalidation: %+v", invalidation)
	}
}

func TestEvaluateSubjectStateChangePropagatesToDependents(t *testing.T) {
	now := time.Date(2026, 9, 27, 4, 10, 0, 0, time.UTC)

	got, err := Evaluate(Snapshot{
		At: now,
		StateBindings: []StateBinding{{
			ID:       "target-state",
			Expected: "version-17",
			Observed: "version-18",
		}},
		Assumptions: []Assumption{
			{
				ID:                    "subject-unchanged",
				RequiresStateBindings: []StateBindingID{"target-state"},
			},
			{
				ID:        "safe-to-act",
				DependsOn: []AssumptionID{"subject-unchanged"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(got.InvalidatedAssumptions) != 2 {
		t.Fatalf("expected base and dependent invalidation, got %+v", got.InvalidatedAssumptions)
	}

	reasons := map[AssumptionID]InvalidationReason{}
	for _, invalidation := range got.Invalidations {
		reasons[invalidation.AssumptionID] = invalidation.Reason
	}
	if reasons["subject-unchanged"] != ReasonSubjectStateChanged {
		t.Fatalf("expected subject-state change, got %q", reasons["subject-unchanged"])
	}
	if reasons["safe-to-act"] != ReasonDependencyInvalid {
		t.Fatalf("expected dependency invalidation, got %q", reasons["safe-to-act"])
	}
}

func TestEvaluateRejectsUnknownStateBindingReference(t *testing.T) {
	now := time.Date(2026, 9, 27, 4, 10, 0, 0, time.UTC)

	_, err := Evaluate(Snapshot{
		At: now,
		Assumptions: []Assumption{{
			ID:                    "safe-to-act",
			RequiresStateBindings: []StateBindingID{"missing-binding"},
		}},
	})
	if err == nil {
		t.Fatal("expected unknown state-binding reference error")
	}
}
