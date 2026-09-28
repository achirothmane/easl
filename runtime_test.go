package easl

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/achirothmane/easl/genesis"
)

type bootstrapVerifier struct {
	fail genesis.FailureCode
}

func (v bootstrapVerifier) maybe(code genesis.FailureCode) error {
	if v.fail == code {
		return errors.New("forced bootstrap verification failure")
	}
	return nil
}

func (v bootstrapVerifier) VerifyAuthenticity(context.Context, genesis.Manifest) error {
	return v.maybe(genesis.FailureAuthenticity)
}
func (v bootstrapVerifier) VerifyTrustRoot(context.Context, genesis.Manifest) error {
	return v.maybe(genesis.FailureTrustRoot)
}
func (v bootstrapVerifier) VerifyAttestation(context.Context, genesis.Manifest) error {
	return v.maybe(genesis.FailureAttestation)
}
func (v bootstrapVerifier) VerifyRevocation(context.Context, genesis.Manifest) error {
	return v.maybe(genesis.FailureRevoked)
}
func (v bootstrapVerifier) VerifyBuildProvenance(context.Context, genesis.Manifest) error {
	return v.maybe(genesis.FailureBuildProvenance)
}
func (v bootstrapVerifier) VerifySpecBuildBinding(context.Context, genesis.Manifest) error {
	return v.maybe(genesis.FailureSpecBuildBinding)
}
func (v bootstrapVerifier) VerifyDefaultDeny(context.Context, genesis.Manifest) error {
	return v.maybe(genesis.FailureDefaultDeny)
}
func (v bootstrapVerifier) VerifyProofRequirements(context.Context, genesis.Manifest) error {
	return v.maybe(genesis.FailureProofRequirements)
}

func TestDirectEvaluateFailsClosedWithoutGenesis(t *testing.T) {
	got, err := Evaluate(Snapshot{At: time.Now().UTC()})
	if !errors.Is(err, ErrGenesisNotReady) {
		t.Fatalf("Evaluate() error = %v, want %v", err, ErrGenesisNotReady)
	}
	if got.State != StateInvalid || got.EvidenceStatus != EvidenceInsufficient {
		t.Fatalf("direct Evaluate() must fail closed, got %+v", got)
	}
}

func TestZeroRuntimeFailsClosed(t *testing.T) {
	var runtime Runtime

	got, err := runtime.Evaluate(Snapshot{At: time.Now().UTC()})
	if !errors.Is(err, ErrGenesisNotReady) {
		t.Fatalf("Runtime.Evaluate() error = %v, want %v", err, ErrGenesisNotReady)
	}
	if got.State != StateInvalid {
		t.Fatalf("zero Runtime must fail closed, got %+v", got)
	}
}

func TestBootstrapRequiresExplicitImplementationBinding(t *testing.T) {
	now := time.Date(2026, 9, 28, 4, 0, 0, 0, time.UTC)
	manifest := bootstrapManifest(now)

	runtime, result := Bootstrap(context.Background(), BootstrapInput{
		Manifest: manifest,
		Verification: genesis.Context{
			Now:                 now,
			RequiredConformance: genesis.ConformanceC3,
		},
		Verifier: bootstrapVerifier{},
	})

	if runtime != nil {
		t.Fatal("expected no runtime without expected implementation digest")
	}
	assertGenesisFailure(t, result, genesis.FailureImplementationMismatch)
}

func TestBootstrapRejectsGenesisRollback(t *testing.T) {
	now := time.Date(2026, 9, 28, 4, 0, 0, 0, time.UTC)
	manifest := bootstrapManifest(now)
	manifest.GenesisEpoch = 6

	runtime, result := Bootstrap(context.Background(), BootstrapInput{
		Manifest: manifest,
		Verification: genesis.Context{
			Now:                          now,
			MinimumAcceptedEpoch:         7,
			ExpectedImplementationDigest: rootDigest("6"),
			RequiredConformance:          genesis.ConformanceC3,
		},
		Verifier: bootstrapVerifier{},
	})

	if runtime != nil {
		t.Fatal("expected no runtime for rolled-back Genesis epoch")
	}
	assertGenesisFailure(t, result, genesis.FailureEpochRollback)
}

func TestBootstrapRejectsExternalVerificationFailure(t *testing.T) {
	now := time.Date(2026, 9, 28, 4, 0, 0, 0, time.UTC)
	manifest := bootstrapManifest(now)

	runtime, result := Bootstrap(context.Background(), BootstrapInput{
		Manifest: manifest,
		Verification: genesis.Context{
			Now:                          now,
			MinimumAcceptedEpoch:         7,
			ExpectedImplementationDigest: rootDigest("6"),
			RequiredConformance:          genesis.ConformanceC3,
		},
		Verifier: bootstrapVerifier{fail: genesis.FailureDefaultDeny},
	})

	if runtime != nil {
		t.Fatal("expected no runtime when default-deny verification fails")
	}
	assertGenesisFailure(t, result, genesis.FailureDefaultDeny)
}

func TestBootstrapCreatesOnlyOperationalEvaluationPath(t *testing.T) {
	now := time.Date(2026, 9, 28, 4, 0, 0, 0, time.UTC)
	manifest := bootstrapManifest(now)

	runtime, result := Bootstrap(context.Background(), BootstrapInput{
		Manifest: manifest,
		Verification: genesis.Context{
			Now:                          now,
			MinimumAcceptedEpoch:         7,
			ExpectedImplementationDigest: rootDigest("6"),
			RequiredConformance:          genesis.ConformanceC3,
		},
		Verifier: bootstrapVerifier{},
	})
	if result.State != genesis.StateReady {
		t.Fatalf("bootstrap state = %s, failures=%v", result.State, result.Failures)
	}
	if runtime == nil {
		t.Fatal("expected operational runtime after BOOTSTRAP_READY")
	}

	evaluation, err := runtime.Evaluate(Snapshot{
		At: now,
		Assumptions: []Assumption{{
			ID: "bootstrap-authorized-evaluation",
		}},
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate() error = %v", err)
	}
	if evaluation.State != StateValid {
		t.Fatalf("expected VALID evaluation, got %+v", evaluation)
	}

	metadata, err := runtime.Metadata()
	if err != nil {
		t.Fatalf("Metadata() error = %v", err)
	}
	if metadata.GenesisEpoch != manifest.GenesisEpoch ||
		metadata.ImplementationDigest != manifest.Implementation.ImplementationDigest ||
		metadata.ConformanceLevel != genesis.ConformanceC3 {
		t.Fatalf("unexpected runtime metadata: %+v", metadata)
	}
}

func assertGenesisFailure(t *testing.T, result genesis.Result, code genesis.FailureCode) {
	t.Helper()
	if result.State != genesis.StateLocked {
		t.Fatalf("state = %s, want %s", result.State, genesis.StateLocked)
	}
	for _, failure := range result.Failures {
		if failure.Code == code {
			return
		}
	}
	t.Fatalf("missing Genesis failure %s in %+v", code, result.Failures)
}

func bootstrapManifest(now time.Time) genesis.Manifest {
	return genesis.Manifest{
		ManifestVersion:     "1.0",
		GenesisEpoch:        7,
		ArchitectureVersion: "level-minus-1/v1.0",
		Specification: genesis.Specification{
			SpecHash:              rootDigest("a"),
			InvariantSetHash:      rootDigest("b"),
			AssumptionSetHash:     rootDigest("c"),
			ForbiddenStateSetHash: rootDigest("d"),
			TauBoundsHash:         rootDigest("e"),
		},
		Verification: genesis.Verification{
			Mode:           "mixed",
			ProofStatus:    "PASS",
			ProofArtifacts: []string{"proof://genesis"},
			Toolchain:      []string{"TLC", "TLAPS"},
			ModelBounds:    []string{"finite-bootstrap-state"},
			ProofScopeHash: rootDigest("f"),
		},
		ThreatModel: genesis.ThreatModel{
			ThreatModelHash:        rootDigest("1"),
			CapabilityEnvelopeHash: rootDigest("2"),
		},
		Trust: genesis.Trust{
			TrustRootRef:          "trust://root/7",
			TrustRootEpoch:        7,
			AttestationPolicyHash: rootDigest("3"),
			NoncePolicyHash:       rootDigest("4"),
			RevocationRef:         "revocation://root/7",
		},
		Enforcement: genesis.Enforcement{
			EnforcementPolicyHash: rootDigest("5"),
			DefaultDenyRequired:   true,
		},
		Implementation: genesis.Implementation{
			ImplementationDigest:   rootDigest("6"),
			RefinementMappingHash:  rootDigest("7"),
			ConformanceLevel:       genesis.ConformanceC3,
			ExecutableContractHash: rootDigest("8"),
		},
		SupplyChain: genesis.SupplyChain{
			SourceRevision:     "git:abc123",
			BuilderIdentity:    "builder://ci",
			BuildProvenanceRef: "provenance://build/1",
			MaterialsHash:      rootDigest("9"),
			SBOMHash:           rootDigest("0"),
		},
		Validity: genesis.Validity{
			BuiltAt:              now.Add(-2 * time.Hour),
			ValidFrom:            now.Add(-time.Hour),
			ValidUntil:           now.Add(time.Hour),
			MinimumAcceptedEpoch: 7,
		},
		Approval: genesis.Approval{
			ApprovedBy:         []string{"security://approver"},
			ApprovalPolicyHash: rootDigest("a"),
		},
		Authenticity: genesis.Authenticity{
			Canonicalization:   "RFC8785",
			SignatureScope:     "MANIFEST_EXCLUDING_AUTHENTICITY",
			SignedPayloadHash:  rootDigest("b"),
			SignatureAlgorithm: "ed25519",
			SignerKeyID:        "key://genesis/7",
			Signature:          "base64:signature",
		},
	}
}

func rootDigest(ch string) string {
	return "sha256:" + strings.Repeat(ch, 64)
}
