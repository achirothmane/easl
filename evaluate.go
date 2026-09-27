package easl

import (
	"fmt"
	"sort"
)

func Evaluate(s Snapshot) (Evaluation, error) {
	if s.At.IsZero() {
		return Evaluation{}, fmt.Errorf("easl: zero evaluation time")
	}

	evidence := map[EvidenceID]Evidence{}
	stale := map[EvidenceID]bool{}
	for _, e := range s.Evidence {
		if e.ID == "" {
			return Evaluation{}, fmt.Errorf("easl: empty evidence id")
		}
		if _, ok := evidence[e.ID]; ok {
			return Evaluation{}, fmt.Errorf("easl: duplicate evidence id %q", e.ID)
		}
		evidence[e.ID] = e
		stale[e.ID] = e.ExpiresAt != nil && !s.At.Before(*e.ExpiresAt)
	}

	stateBindings := map[StateBindingID]StateBinding{}
	for _, binding := range s.StateBindings {
		if binding.ID == "" {
			return Evaluation{}, fmt.Errorf("easl: empty state binding id")
		}
		if binding.Expected == "" || binding.Observed == "" {
			return Evaluation{}, fmt.Errorf("easl: state binding %q requires expected and observed tokens", binding.ID)
		}
		if _, ok := stateBindings[binding.ID]; ok {
			return Evaluation{}, fmt.Errorf("easl: duplicate state binding id %q", binding.ID)
		}
		stateBindings[binding.ID] = binding
	}

	assumptions := map[AssumptionID]Assumption{}
	for _, a := range s.Assumptions {
		if a.ID == "" {
			return Evaluation{}, fmt.Errorf("easl: empty assumption id")
		}
		if _, ok := assumptions[a.ID]; ok {
			return Evaluation{}, fmt.Errorf("easl: duplicate assumption id %q", a.ID)
		}
		assumptions[a.ID] = a
	}
	if err := validateGraph(assumptions); err != nil {
		return Evaluation{}, err
	}

	invalid := map[AssumptionID]bool{}
	var contradictions []Contradiction
	var invalidations []Invalidation
	for _, e := range s.Evidence {
		if stale[e.ID] {
			continue
		}
		for _, target := range e.Contradicts {
			if _, ok := assumptions[target]; !ok {
				return Evaluation{}, fmt.Errorf("easl: evidence %q contradicts unknown assumption %q", e.ID, target)
			}
			invalid[target] = true
			contradictions = append(contradictions, Contradiction{EvidenceID: e.ID, AssumptionID: target})
			invalidations = append(invalidations, Invalidation{AssumptionID: target, Reason: ReasonContradicted, EvidenceID: e.ID})
		}
	}

	for _, a := range s.Assumptions {
		if a.ValidUntil != nil && !s.At.Before(*a.ValidUntil) {
			invalid[a.ID] = true
			invalidations = append(invalidations, Invalidation{
				AssumptionID: a.ID,
				Reason:       ReasonAssumptionExpired,
			})
		}
		for _, id := range a.Requires {
			if _, ok := evidence[id]; !ok {
				invalid[a.ID] = true
				invalidations = append(invalidations, Invalidation{AssumptionID: a.ID, Reason: ReasonMissingEvidence, EvidenceID: id})
			} else if stale[id] {
				invalid[a.ID] = true
				invalidations = append(invalidations, Invalidation{AssumptionID: a.ID, Reason: ReasonStaleEvidence, EvidenceID: id})
			}
		}
		for _, id := range a.RequiresStateBindings {
			binding, ok := stateBindings[id]
			if !ok {
				return Evaluation{}, fmt.Errorf("easl: assumption %q requires unknown state binding %q", a.ID, id)
			}
			if binding.Expected != binding.Observed {
				invalid[a.ID] = true
				invalidations = append(invalidations, Invalidation{
					AssumptionID:   a.ID,
					Reason:         ReasonSubjectStateChanged,
					StateBindingID: id,
				})
			}
		}
	}

	for changed := true; changed; {
		changed = false
		for _, a := range s.Assumptions {
			if invalid[a.ID] {
				continue
			}
			for _, dep := range a.DependsOn {
				if invalid[dep] {
					invalid[a.ID] = true
					invalidations = append(invalidations, Invalidation{AssumptionID: a.ID, Reason: ReasonDependencyInvalid, DependencyID: dep})
					changed = true
					break
				}
			}
		}
	}

	out := Evaluation{State: StateValid, EvidenceStatus: EvidenceSufficient, Contradictions: contradictions, Invalidations: invalidations}
	for id, isStale := range stale {
		if isStale {
			out.StaleEvidence = append(out.StaleEvidence, id)
		}
	}
	for id := range invalid {
		out.InvalidatedAssumptions = append(out.InvalidatedAssumptions, id)
	}
	sort.Slice(out.StaleEvidence, func(i, j int) bool { return out.StaleEvidence[i] < out.StaleEvidence[j] })
	sort.Slice(out.InvalidatedAssumptions, func(i, j int) bool { return out.InvalidatedAssumptions[i] < out.InvalidatedAssumptions[j] })

	if len(out.Contradictions) > 0 {
		out.State, out.EvidenceStatus = StateInvalid, EvidenceContradictory
	} else if len(out.InvalidatedAssumptions) > 0 {
		out.State, out.EvidenceStatus = StateInvalid, EvidenceInsufficient
	} else if len(out.StaleEvidence) > 0 {
		out.State = StateDegraded
	}
	return out, nil
}

func validateGraph(all map[AssumptionID]Assumption) error {
	state := map[AssumptionID]uint8{}
	var visit func(AssumptionID) error
	visit = func(id AssumptionID) error {
		if state[id] == 1 {
			return fmt.Errorf("easl: dependency cycle at %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, dep := range all[id].DependsOn {
			if _, ok := all[dep]; !ok {
				return fmt.Errorf("easl: assumption %q depends on unknown assumption %q", id, dep)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range all {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
