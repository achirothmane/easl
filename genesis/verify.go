package genesis

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

type State string

const (
	StateLocked State = "GENESIS_LOCKED"
	StateReady  State = "BOOTSTRAP_READY"
)

type FailureCode string

const (
	FailureMalformedManifest       FailureCode = "MALFORMED_MANIFEST"
	FailureManifestNotYetValid     FailureCode = "MANIFEST_NOT_YET_VALID"
	FailureManifestExpired         FailureCode = "MANIFEST_EXPIRED"
	FailureEpochRollback           FailureCode = "EPOCH_ROLLBACK"
	FailureDoctrineRollback        FailureCode = "DOCTRINE_EPOCH_ROLLBACK"
	FailureDoctrineMismatch        FailureCode = "DOCTRINE_MANIFEST_MISMATCH"
	FailureDoctrineUnverified      FailureCode = "DOCTRINE_UNVERIFIED"
	FailureProofRequirements       FailureCode = "PROOF_REQUIREMENTS_UNSATISFIED"
	FailureSpecBuildBinding        FailureCode = "SPEC_BUILD_BINDING_UNVERIFIED"
	FailureBuildProvenance         FailureCode = "BUILD_PROVENANCE_UNVERIFIED"
	FailureTrustRoot               FailureCode = "TRUST_ROOT_UNVERIFIED"
	FailureAttestation             FailureCode = "ATTESTATION_UNVERIFIED"
	FailureRevoked                 FailureCode = "MANIFEST_OR_TRUST_REVOKED"
	FailureDefaultDeny             FailureCode = "DEFAULT_DENY_INACTIVE"
	FailureAuthenticity            FailureCode = "MANIFEST_AUTHENTICITY_UNVERIFIED"
	FailureImplementationMismatch  FailureCode = "IMPLEMENTATION_DIGEST_MISMATCH"
	FailureConformanceInsufficient FailureCode = "CONFORMANCE_LEVEL_INSUFFICIENT"
	FailureVerifierUnavailable     FailureCode = "EXTERNAL_VERIFIER_UNAVAILABLE"
)

type Failure struct {
	Code   FailureCode `json:"code"`
	Detail string      `json:"detail,omitempty"`
}

type Result struct {
	State    State     `json:"state"`
	Failures []Failure `json:"failures,omitempty"`
}

type Context struct {
	Now                          time.Time
	MinimumAcceptedEpoch         uint64
	MinimumAcceptedDoctrineEpoch uint64
	ExpectedDoctrineManifestHash string
	ExpectedImplementationDigest string
	RequiredConformance          ConformanceLevel
}

// ExternalVerifier binds the declarative manifest to facts that cannot be
// established from the JSON document alone.
type ExternalVerifier interface {
	VerifyDoctrineBinding(context.Context, Manifest) error
	VerifyAuthenticity(context.Context, Manifest) error
	VerifyTrustRoot(context.Context, Manifest) error
	VerifyAttestation(context.Context, Manifest) error
	VerifyRevocation(context.Context, Manifest) error
	VerifyBuildProvenance(context.Context, Manifest) error
	VerifySpecBuildBinding(context.Context, Manifest) error
	VerifyDefaultDeny(context.Context, Manifest) error
	VerifyProofRequirements(context.Context, Manifest) error
}

// Verify evaluates NO_BOOTSTRAP_WITHOUT_VALID_GENESIS. Every missing or failed
// requirement produces GENESIS_LOCKED. No permissive fallback exists.
func Verify(ctx context.Context, m Manifest, c Context, external ExternalVerifier) Result {
	var failures []Failure
	add := func(code FailureCode, detail string) {
		failures = append(failures, Failure{Code: code, Detail: detail})
	}

	if c.Now.IsZero() {
		add(FailureMalformedManifest, "verification time is zero")
	}
	if err := validateRequiredFields(m); err != nil {
		add(FailureMalformedManifest, err.Error())
	}
	if !m.Validity.ValidFrom.IsZero() && c.Now.Before(m.Validity.ValidFrom) {
		add(FailureManifestNotYetValid, "manifest validity window has not started")
	}
	if !m.Validity.ValidUntil.IsZero() && !c.Now.Before(m.Validity.ValidUntil) {
		add(FailureManifestExpired, "manifest validity window has ended")
	}

	minEpoch := c.MinimumAcceptedEpoch
	if m.Validity.MinimumAcceptedEpoch > minEpoch {
		minEpoch = m.Validity.MinimumAcceptedEpoch
	}
	if m.GenesisEpoch < minEpoch {
		add(FailureEpochRollback, fmt.Sprintf("genesis epoch %d is below minimum %d", m.GenesisEpoch, minEpoch))
	}
	if m.Doctrine.DoctrineEpoch < c.MinimumAcceptedDoctrineEpoch {
		add(FailureDoctrineRollback, fmt.Sprintf("doctrine epoch %d is below minimum %d", m.Doctrine.DoctrineEpoch, c.MinimumAcceptedDoctrineEpoch))
	}
	if c.ExpectedDoctrineManifestHash != "" && m.Doctrine.DoctrineManifestHash != c.ExpectedDoctrineManifestHash {
		add(FailureDoctrineMismatch, "doctrine manifest hash does not match the expected active doctrine")
	}

	if m.Verification.ProofStatus != "PASS" {
		add(FailureProofRequirements, "proof_status must be PASS")
	}
	if !m.Enforcement.DefaultDenyRequired {
		add(FailureDefaultDeny, "manifest does not require default-deny enforcement")
	}
	if c.ExpectedImplementationDigest != "" && m.Implementation.ImplementationDigest != c.ExpectedImplementationDigest {
		add(FailureImplementationMismatch, "implementation digest does not match the expected runtime artifact")
	}
	if !conformanceAtLeast(m.Implementation.ConformanceLevel, c.RequiredConformance) {
		add(FailureConformanceInsufficient, fmt.Sprintf("conformance %s is below required %s", m.Implementation.ConformanceLevel, c.RequiredConformance))
	}

	if external == nil {
		add(FailureVerifierUnavailable, "external verifier is required")
		return locked(failures)
	}

	checks := []struct {
		code FailureCode
		fn   func(context.Context, Manifest) error
	}{
		{FailureDoctrineUnverified, external.VerifyDoctrineBinding},
		{FailureAuthenticity, external.VerifyAuthenticity},
		{FailureTrustRoot, external.VerifyTrustRoot},
		{FailureAttestation, external.VerifyAttestation},
		{FailureRevoked, external.VerifyRevocation},
		{FailureBuildProvenance, external.VerifyBuildProvenance},
		{FailureSpecBuildBinding, external.VerifySpecBuildBinding},
		{FailureDefaultDeny, external.VerifyDefaultDeny},
		{FailureProofRequirements, external.VerifyProofRequirements},
	}
	for _, check := range checks {
		if err := check.fn(ctx, m); err != nil {
			add(check.code, err.Error())
		}
	}

	if len(failures) != 0 {
		return locked(failures)
	}
	return Result{State: StateReady}
}

func locked(failures []Failure) Result {
	return Result{State: StateLocked, Failures: failures}
}

func validateRequiredFields(m Manifest) error {
	required := map[string]string{
		"manifest_version":         m.ManifestVersion,
		"architecture_version":     m.ArchitectureVersion,
		"doctrine_id":              m.Doctrine.DoctrineID,
		"doctrine_manifest_hash":   m.Doctrine.DoctrineManifestHash,
		"spec_hash":                m.Specification.SpecHash,
		"invariant_set_hash":       m.Specification.InvariantSetHash,
		"assumption_set_hash":      m.Specification.AssumptionSetHash,
		"forbidden_state_set_hash": m.Specification.ForbiddenStateSetHash,
		"tau_bounds_hash":          m.Specification.TauBoundsHash,
		"proof_scope_hash":         m.Verification.ProofScopeHash,
		"threat_model_hash":        m.ThreatModel.ThreatModelHash,
		"capability_envelope_hash": m.ThreatModel.CapabilityEnvelopeHash,
		"trust_root_ref":           m.Trust.TrustRootRef,
		"attestation_policy_hash":  m.Trust.AttestationPolicyHash,
		"nonce_policy_hash":        m.Trust.NoncePolicyHash,
		"revocation_ref":           m.Trust.RevocationRef,
		"enforcement_policy_hash":  m.Enforcement.EnforcementPolicyHash,
		"implementation_digest":    m.Implementation.ImplementationDigest,
		"refinement_mapping_hash":  m.Implementation.RefinementMappingHash,
		"executable_contract_hash": m.Implementation.ExecutableContractHash,
		"source_revision":          m.SupplyChain.SourceRevision,
		"builder_identity":         m.SupplyChain.BuilderIdentity,
		"build_provenance_ref":     m.SupplyChain.BuildProvenanceRef,
		"materials_hash":           m.SupplyChain.MaterialsHash,
		"sbom_hash":                m.SupplyChain.SBOMHash,
		"approval_policy_hash":     m.Approval.ApprovalPolicyHash,
		"canonicalization":         m.Authenticity.Canonicalization,
		"signed_payload_hash":      m.Authenticity.SignedPayloadHash,
		"signature_algorithm":      m.Authenticity.SignatureAlgorithm,
		"signer_key_id":            m.Authenticity.SignerKeyID,
		"signature":                m.Authenticity.Signature,
	}
	for name, value := range required {
		if value == "" {
			return fmt.Errorf("missing required field %s", name)
		}
	}
	if m.Sequence > 0 && m.PreviousManifestHash == "" {
		return fmt.Errorf("previous_manifest_hash is required when sequence > 0")
	}
	if m.Validity.BuiltAt.IsZero() || m.Validity.ValidFrom.IsZero() || m.Validity.ValidUntil.IsZero() {
		return fmt.Errorf("built_at, valid_from, and valid_until are required")
	}
	if !m.Validity.ValidFrom.Before(m.Validity.ValidUntil) {
		return fmt.Errorf("valid_from must be before valid_until")
	}
	if len(m.Approval.ApprovedBy) == 0 {
		return fmt.Errorf("approved_by must not be empty")
	}

	if m.Authenticity.Canonicalization != "RFC8785" {
		return fmt.Errorf("canonicalization must be RFC8785")
	}
	if m.Authenticity.SignatureScope != "MANIFEST_EXCLUDING_AUTHENTICITY" {
		return fmt.Errorf("signature_scope must be MANIFEST_EXCLUDING_AUTHENTICITY")
	}
	hashes := map[string]string{
		"previous_manifest_hash":   m.PreviousManifestHash,
		"doctrine_manifest_hash":   m.Doctrine.DoctrineManifestHash,   m.PreviousManifestHash,
		"spec_hash":                m.Specification.SpecHash,
		"invariant_set_hash":       m.Specification.InvariantSetHash,
		"assumption_set_hash":      m.Specification.AssumptionSetHash,
		"forbidden_state_set_hash": m.Specification.ForbiddenStateSetHash,
		"tau_bounds_hash":          m.Specification.TauBoundsHash,
		"proof_scope_hash":         m.Verification.ProofScopeHash,
		"threat_model_hash":        m.ThreatModel.ThreatModelHash,
		"capability_envelope_hash": m.ThreatModel.CapabilityEnvelopeHash,
		"attestation_policy_hash":  m.Trust.AttestationPolicyHash,
		"nonce_policy_hash":        m.Trust.NoncePolicyHash,
		"enforcement_policy_hash":  m.Enforcement.EnforcementPolicyHash,
		"implementation_digest":    m.Implementation.ImplementationDigest,
		"refinement_mapping_hash":  m.Implementation.RefinementMappingHash,
		"executable_contract_hash": m.Implementation.ExecutableContractHash,
		"materials_hash":           m.SupplyChain.MaterialsHash,
		"sbom_hash":                m.SupplyChain.SBOMHash,
		"approval_policy_hash":     m.Approval.ApprovalPolicyHash,
		"signed_payload_hash":      m.Authenticity.SignedPayloadHash,
	}
	for name, value := range hashes {
		if name == "previous_manifest_hash" && value == "" && m.Sequence == 0 {
			continue
		}
		if !sha256Digest.MatchString(value) {
			return fmt.Errorf("%s must be a sha256 digest", name)
		}
	}
	if _, ok := conformanceRank[m.Implementation.ConformanceLevel]; !ok {
		return fmt.Errorf("unknown conformance level %q", m.Implementation.ConformanceLevel)
	}
	switch m.Verification.Mode {
	case "model_checked", "theorem_proved", "mixed":
	default:
		return fmt.Errorf("unknown verification_mode %q", m.Verification.Mode)
	}
	return nil
}

var sha256Digest = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

var conformanceRank = map[ConformanceLevel]int{
	ConformanceC0: 0,
	ConformanceC1: 1,
	ConformanceC2: 2,
	ConformanceC3: 3,
	ConformanceC4: 4,
}

func conformanceAtLeast(actual, required ConformanceLevel) bool {
	if required == "" {
		return true
	}
	a, aok := conformanceRank[actual]
	r, rok := conformanceRank[required]
	return aok && rok && a >= r
}
