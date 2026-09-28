package genesis

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeVerifier struct {
	fail FailureCode
}

func (f fakeVerifier) err(code FailureCode) error {
	if f.fail == code {
		return errors.New("forced verification failure")
	}
	return nil
}

func (f fakeVerifier) VerifyDoctrineBinding(context.Context, Manifest) error {
	return f.err(FailureDoctrineUnverified)
}
func (f fakeVerifier) VerifyAuthenticity(context.Context, Manifest) error {
	return f.err(FailureAuthenticity)
}
func (f fakeVerifier) VerifyTrustRoot(context.Context, Manifest) error {
	return f.err(FailureTrustRoot)
}
func (f fakeVerifier) VerifyAttestation(context.Context, Manifest) error {
	return f.err(FailureAttestation)
}
func (f fakeVerifier) VerifyRevocation(context.Context, Manifest) error {
	return f.err(FailureRevoked)
}
func (f fakeVerifier) VerifyBuildProvenance(context.Context, Manifest) error {
	return f.err(FailureBuildProvenance)
}
func (f fakeVerifier) VerifySpecBuildBinding(context.Context, Manifest) error {
	return f.err(FailureSpecBuildBinding)
}
func (f fakeVerifier) VerifyDefaultDeny(context.Context, Manifest) error {
	return f.err(FailureDefaultDeny)
}
func (f fakeVerifier) VerifyProofRequirements(context.Context, Manifest) error {
	return f.err(FailureProofRequirements)
}

func validManifest(now time.Time) Manifest {
	return Manifest{
		ManifestVersion:     "1.1",
		GenesisEpoch:        7,
		Sequence:            0,
		ArchitectureVersion: "level-minus-1/v1.1",
		Doctrine: DoctrineBinding{
			DoctrineID:           "aegis-ege-doctrine",
			DoctrineEpoch:        3,
			DoctrineManifestHash: digest("d"),
		},
		Specification: Specification{
			SpecHash:              digest("a"),
			InvariantSetHash:      digest("b"),
			AssumptionSetHash:     digest("c"),
			ForbiddenStateSetHash: digest("d"),
			TauBoundsHash:         digest("e"),
		},
		Verification: Verification{
			Mode:           "mixed",
			ProofStatus:    "PASS",
			ProofArtifacts: []string{"proof://genesis"},
			Toolchain:      []string{"TLC", "TLAPS"},
			ModelBounds:    []string{"finite-bootstrap-state"},
			ProofScopeHash: digest("f"),
		},
		ThreatModel: ThreatModel{
			ThreatModelHash:        digest("1"),
			CapabilityEnvelopeHash: digest("2"),
		},
		Trust: Trust{
			TrustRootRef:          "trust://root/7",
			TrustRootEpoch:        7,
			AttestationPolicyHash: digest("3"),
			NoncePolicyHash:       digest("4"),
			RevocationRef:         "revocation://root/7",
		},
		Enforcement: Enforcement{
			EnforcementPolicyHash: digest("5"),
			DefaultDenyRequired:   true,
		},
		Implementation: Implementation{
			ImplementationDigest:   digest("6"),
			RefinementMappingHash:  digest("7"),
			ConformanceLevel:       ConformanceC3,
			ExecutableContractHash: digest("8"),
		},
		SupplyChain: SupplyChain{
			SourceRevision:     "git:abc123",
			BuilderIdentity:    "builder://ci",
			BuildProvenanceRef: "provenance://build/1",
			MaterialsHash:      digest("9"),
			SBOMHash:           digest("0"),
		},
		Validity: Validity{
			BuiltAt:              now.Add(-2 * time.Hour),
			ValidFrom:            now.Add(-time.Hour),
			ValidUntil:           now.Add(time.Hour),
			MinimumAcceptedEpoch: 7,
		},
		Approval: Approval{
			ApprovedBy:         []string{"security://approver"},
			ApprovalPolicyHash: digest("a"),
		},
		Authenticity: Authenticity{
			Canonicalization:   "RFC8785",
			SignatureScope:     "MANIFEST_EXCLUDING_AUTHENTICITY",
			SignedPayloadHash:  digest("b"),
			SignatureAlgorithm: "ed25519",
			SignerKeyID:        "key://genesis/7",
			Signature:          "base64:signature",
		},
	}
}

func digest(ch string) string {
	return "sha256:" + strings.Repeat(ch, 64)
}

func TestVerifyAllowsValidGenesis(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 30, 0, 0, time.UTC)
	m := validManifest(now)

	got := Verify(context.Background(), m, Context{
		Now:                          now,
		MinimumAcceptedEpoch:         7,
		MinimumAcceptedDoctrineEpoch: 3,
		ExpectedDoctrineManifestHash: digest("d"),
		ExpectedImplementationDigest: digest("6"),
		RequiredConformance:          ConformanceC3,
	}, fakeVerifier{})

	if got.State != StateReady {
		t.Fatalf("state = %s, want %s; failures=%v", got.State, StateReady, got.Failures)
	}
}


func TestVerifyFailsClosedOnDoctrineRollback(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 30, 0, 0, time.UTC)
	m := validManifest(now)
	m.Doctrine.DoctrineEpoch = 2

	got := Verify(context.Background(), m, Context{
		Now:                          now,
		MinimumAcceptedDoctrineEpoch: 3,
		ExpectedDoctrineManifestHash: digest("d"),
	}, fakeVerifier{})

	assertFailure(t, got, FailureDoctrineRollback)
}

func TestVerifyFailsClosedOnDoctrineHashMismatch(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 30, 0, 0, time.UTC)
	m := validManifest(now)

	got := Verify(context.Background(), m, Context{
		Now:                          now,
		ExpectedDoctrineManifestHash: digest("e"),
	}, fakeVerifier{})

	assertFailure(t, got, FailureDoctrineMismatch)
}

func TestVerifyFailsClosedWhenDoctrineCannotBeExternallyVerified(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 30, 0, 0, time.UTC)
	m := validManifest(now)

	got := Verify(context.Background(), m, Context{
		Now:                          now,
		ExpectedDoctrineManifestHash: digest("d"),
	}, fakeVerifier{fail: FailureDoctrineUnverified})

	assertFailure(t, got, FailureDoctrineUnverified)
}

func TestVerifyFailsClosedOnRollback(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 30, 0, 0, time.UTC)
	m := validManifest(now)
	m.GenesisEpoch = 6

	got := Verify(context.Background(), m, Context{
		Now:                  now,
		MinimumAcceptedEpoch: 7,
	}, fakeVerifier{})

	assertFailure(t, got, FailureEpochRollback)
}

func TestVerifyFailsClosedOnExpiredManifest(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 30, 0, 0, time.UTC)
	m := validManifest(now)
	m.Validity.ValidUntil = now

	got := Verify(context.Background(), m, Context{Now: now}, fakeVerifier{})

	assertFailure(t, got, FailureManifestExpired)
}

func TestVerifyFailsClosedWhenDefaultDenyIsNotActive(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 30, 0, 0, time.UTC)
	m := validManifest(now)

	got := Verify(context.Background(), m, Context{Now: now}, fakeVerifier{fail: FailureDefaultDeny})

	assertFailure(t, got, FailureDefaultDeny)
}

func TestVerifyFailsClosedWhenVerifierUnavailable(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 30, 0, 0, time.UTC)
	m := validManifest(now)

	got := Verify(context.Background(), m, Context{Now: now}, nil)

	assertFailure(t, got, FailureVerifierUnavailable)
}

func TestVerifyRejectsInsufficientConformance(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 30, 0, 0, time.UTC)
	m := validManifest(now)
	m.Implementation.ConformanceLevel = ConformanceC2

	got := Verify(context.Background(), m, Context{
		Now:                 now,
		RequiredConformance: ConformanceC3,
	}, fakeVerifier{})

	assertFailure(t, got, FailureConformanceInsufficient)
}

func assertFailure(t *testing.T, got Result, code FailureCode) {
	t.Helper()
	if got.State != StateLocked {
		t.Fatalf("state = %s, want %s", got.State, StateLocked)
	}
	for _, failure := range got.Failures {
		if failure.Code == code {
			return
		}
	}
	t.Fatalf("missing failure %s in %#v", code, got.Failures)
}
