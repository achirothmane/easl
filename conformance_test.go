package easl

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

type subjectStateConformanceFile struct {
	Version   int                           `json:"version"`
	Primitive string                        `json:"primitive"`
	Cases     []subjectStateConformanceCase `json:"cases"`
}

type subjectStateConformanceCase struct {
	Name                    string           `json:"name"`
	Bindings                []StateBinding   `json:"bindings"`
	Required                []StateBindingID `json:"required"`
	WantState               State            `json:"want_state"`
	WantEvidenceStatus      EvidenceStatus   `json:"want_evidence_status"`
	WantInvalidatedBindings []StateBindingID `json:"want_invalidated_bindings"`
	WantErrorContains       string           `json:"want_error_contains"`
}

func TestSubjectStateBindingConformanceVectors(t *testing.T) {
	raw, err := os.ReadFile("conformance/subject_state_binding.json")
	if err != nil {
		t.Fatalf("read conformance vectors: %v", err)
	}

	var vectors subjectStateConformanceFile
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatalf("decode conformance vectors: %v", err)
	}
	if vectors.Version != 1 {
		t.Fatalf("unsupported conformance version %d", vectors.Version)
	}
	if vectors.Primitive != "subject_state_binding" {
		t.Fatalf("unexpected primitive %q", vectors.Primitive)
	}

	at := time.Date(2026, 9, 27, 13, 30, 0, 0, time.UTC)
	for _, tc := range vectors.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			got, err := evaluateSnapshot(Snapshot{
				At:            at,
				StateBindings: tc.Bindings,
				Assumptions: []Assumption{{
					ID:                    "safe-to-act",
					RequiresStateBindings: tc.Required,
				}},
			})

			if tc.WantErrorContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.WantErrorContains)
				}
				if !strings.Contains(err.Error(), tc.WantErrorContains) {
					t.Fatalf("expected error containing %q, got %q", tc.WantErrorContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got.State != tc.WantState {
				t.Fatalf("state: want %s, got %s", tc.WantState, got.State)
			}
			if got.EvidenceStatus != tc.WantEvidenceStatus {
				t.Fatalf(
					"evidence status: want %s, got %s",
					tc.WantEvidenceStatus,
					got.EvidenceStatus,
				)
			}

			var bindingIDs []StateBindingID
			for _, invalidation := range got.Invalidations {
				if invalidation.AssumptionID == "safe-to-act" &&
					invalidation.Reason == ReasonSubjectStateChanged {
					bindingIDs = append(bindingIDs, invalidation.StateBindingID)
				}
			}
			sort.Slice(bindingIDs, func(i, j int) bool { return bindingIDs[i] < bindingIDs[j] })

			want := append([]StateBindingID(nil), tc.WantInvalidatedBindings...)
			sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })

			if len(bindingIDs) != len(want) {
				t.Fatalf("invalidated bindings: want %v, got %v", want, bindingIDs)
			}
			for i := range want {
				if bindingIDs[i] != want[i] {
					t.Fatalf("invalidated bindings: want %v, got %v", want, bindingIDs)
				}
			}
		})
	}
}
