package easl

import (
	"context"
	"errors"

	"github.com/achirothmane/easl/genesis"
)

// ErrGenesisNotReady is returned whenever epistemic evaluation is attempted
// without a Runtime created by a successful Genesis bootstrap.
var ErrGenesisNotReady = errors.New("easl: genesis bootstrap is not ready")

// BootstrapInput contains the Level -1 material required to create an
// operational EASL runtime.
type BootstrapInput struct {
	Manifest     genesis.Manifest
	Verification genesis.Context
	Verifier     genesis.ExternalVerifier
}

// Runtime is the only operational entry point for EASL evaluation.
//
// Its readiness state is intentionally unexported. External callers cannot
// construct a ready Runtime directly; Bootstrap is the only constructor that
// can set it after genesis.Verify returns BOOTSTRAP_READY.
type Runtime struct {
	ready                bool
	genesisEpoch         uint64
	manifestVersion      string
	implementationDigest string
	conformanceLevel     genesis.ConformanceLevel
}

// RuntimeMetadata exposes the Genesis identity bound to an operational runtime
// without exposing mutable readiness state.
type RuntimeMetadata struct {
	GenesisEpoch         uint64                   `json:"genesis_epoch"`
	ManifestVersion      string                   `json:"manifest_version"`
	ImplementationDigest string                   `json:"implementation_digest"`
	ConformanceLevel     genesis.ConformanceLevel `json:"conformance_level"`
}

// Bootstrap enforces NO_BOOTSTRAP_WITHOUT_VALID_GENESIS.
//
// It returns a Runtime only when the Genesis verifier reaches BOOTSTRAP_READY.
// The operational EASL path additionally requires an explicit expected
// implementation digest and minimum conformance level so a caller cannot
// accidentally bootstrap with an unbound build.
func Bootstrap(ctx context.Context, in BootstrapInput) (*Runtime, genesis.Result) {
	if in.Verification.ExpectedImplementationDigest == "" {
		return nil, genesis.Result{
			State: genesis.StateLocked,
			Failures: []genesis.Failure{{
				Code:   genesis.FailureImplementationMismatch,
				Detail: "expected implementation digest is required for operational bootstrap",
			}},
		}
	}
	if in.Verification.RequiredConformance == "" {
		return nil, genesis.Result{
			State: genesis.StateLocked,
			Failures: []genesis.Failure{{
				Code:   genesis.FailureConformanceInsufficient,
				Detail: "required conformance level is required for operational bootstrap",
			}},
		}
	}

	result := genesis.Verify(ctx, in.Manifest, in.Verification, in.Verifier)
	if result.State != genesis.StateReady {
		return nil, result
	}

	return &Runtime{
		ready:                true,
		genesisEpoch:         in.Manifest.GenesisEpoch,
		manifestVersion:      in.Manifest.ManifestVersion,
		implementationDigest: in.Manifest.Implementation.ImplementationDigest,
		conformanceLevel:     in.Manifest.Implementation.ConformanceLevel,
	}, result
}

// Evaluate computes epistemic state only after successful Genesis bootstrap.
func (r *Runtime) Evaluate(s Snapshot) (Evaluation, error) {
	if r == nil || !r.ready {
		return lockedEvaluation(), ErrGenesisNotReady
	}
	return evaluateSnapshot(s)
}

// Metadata returns the Genesis identity bound to this Runtime.
func (r *Runtime) Metadata() (RuntimeMetadata, error) {
	if r == nil || !r.ready {
		return RuntimeMetadata{}, ErrGenesisNotReady
	}
	return RuntimeMetadata{
		GenesisEpoch:         r.genesisEpoch,
		ManifestVersion:      r.manifestVersion,
		ImplementationDigest: r.implementationDigest,
		ConformanceLevel:     r.conformanceLevel,
	}, nil
}

func lockedEvaluation() Evaluation {
	return Evaluation{
		State:          StateInvalid,
		EvidenceStatus: EvidenceInsufficient,
	}
}
