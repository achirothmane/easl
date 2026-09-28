package genesis

import "time"

// ConformanceLevel expresses how strongly an implementation is bound to the
// verified specification. Higher levels include the guarantees of lower levels.
type ConformanceLevel string

const (
	ConformanceC0 ConformanceLevel = "C0"
	ConformanceC1 ConformanceLevel = "C1"
	ConformanceC2 ConformanceLevel = "C2"
	ConformanceC3 ConformanceLevel = "C3"
	ConformanceC4 ConformanceLevel = "C4"
)

// Manifest is the cryptographically bound Level -1 output consumed before
// EASL bootstrap. Domain payloads stay opaque; the manifest only carries
// identities, assurance state, provenance, validity, and policy bindings.
type Manifest struct {
	ManifestVersion      string `json:"manifest_version"`
	GenesisEpoch         uint64 `json:"genesis_epoch"`
	Sequence             uint64 `json:"sequence"`
	PreviousManifestHash string `json:"previous_manifest_hash,omitempty"`
	ArchitectureVersion  string `json:"architecture_version"`

	Specification  Specification  `json:"specification"`
	Verification   Verification   `json:"verification"`
	ThreatModel    ThreatModel    `json:"threat_model"`
	Trust          Trust          `json:"trust"`
	Enforcement    Enforcement    `json:"enforcement"`
	Implementation Implementation `json:"implementation"`
	SupplyChain    SupplyChain    `json:"supply_chain"`
	Validity       Validity       `json:"validity"`
	Approval       Approval       `json:"approval"`
	Authenticity   Authenticity   `json:"authenticity"`
}

type Specification struct {
	SpecHash              string `json:"spec_hash"`
	InvariantSetHash      string `json:"invariant_set_hash"`
	AssumptionSetHash     string `json:"assumption_set_hash"`
	ForbiddenStateSetHash string `json:"forbidden_state_set_hash"`
	TauBoundsHash         string `json:"tau_bounds_hash"`
}

type Verification struct {
	Mode           string   `json:"verification_mode"`
	ProofStatus    string   `json:"proof_status"`
	ProofArtifacts []string `json:"proof_artifacts"`
	Toolchain      []string `json:"toolchain"`
	ModelBounds    []string `json:"model_bounds"`
	ProofScopeHash string   `json:"proof_scope_hash"`
}

type ThreatModel struct {
	ThreatModelHash        string `json:"threat_model_hash"`
	CapabilityEnvelopeHash string `json:"capability_envelope_hash"`
}

type Trust struct {
	TrustRootRef          string `json:"trust_root_ref"`
	TrustRootEpoch        uint64 `json:"trust_root_epoch"`
	AttestationPolicyHash string `json:"attestation_policy_hash"`
	NoncePolicyHash       string `json:"nonce_policy_hash"`
	RevocationRef         string `json:"revocation_ref"`
}

type Enforcement struct {
	EnforcementPolicyHash string `json:"enforcement_policy_hash"`
	DefaultDenyRequired   bool   `json:"default_deny_required"`
}

type Implementation struct {
	ImplementationDigest   string           `json:"implementation_digest"`
	RefinementMappingHash  string           `json:"refinement_mapping_hash"`
	ConformanceLevel       ConformanceLevel `json:"conformance_level"`
	ExecutableContractHash string           `json:"executable_contract_hash"`
}

type SupplyChain struct {
	SourceRevision     string `json:"source_revision"`
	BuilderIdentity    string `json:"builder_identity"`
	BuildProvenanceRef string `json:"build_provenance_ref"`
	MaterialsHash      string `json:"materials_hash"`
	SBOMHash           string `json:"sbom_hash"`
}

type Validity struct {
	BuiltAt              time.Time `json:"built_at"`
	ValidFrom            time.Time `json:"valid_from"`
	ValidUntil           time.Time `json:"valid_until"`
	MinimumAcceptedEpoch uint64    `json:"minimum_accepted_epoch"`
}

type Approval struct {
	ApprovedBy         []string `json:"approved_by"`
	ApprovalPolicyHash string   `json:"approval_policy_hash"`
}

type Authenticity struct {
	Canonicalization   string `json:"canonicalization"`
	SignatureScope     string `json:"signature_scope"`
	SignedPayloadHash  string `json:"signed_payload_hash"`
	SignatureAlgorithm string `json:"signature_algorithm"`
	SignerKeyID        string `json:"signer_key_id"`
	Signature          string `json:"signature"`
}
