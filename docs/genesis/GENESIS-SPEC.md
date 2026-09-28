# Level -1 — Meta-Architectural Genesis & Formal Assurance

Status: **Baseline v1.0**

## Purpose

Level -1 exists before operational bootstrap. It does not authorize domain actions and it does not claim that a formal model proves every property of the real world. Its job is narrower and stronger: produce a cryptographically verifiable **Genesis Manifest** that binds the accepted specification, threat model, trust roots, enforcement policy, implementation/build identity, proof scope, provenance, validity window, and approval policy into one pre-bootstrap decision surface.

The governing rule is:

```text
NO_BOOTSTRAP_WITHOUT_VALID_GENESIS
```

EASL, its initial state, and the first execution intent are not operationally legitimate until the Genesis gate succeeds.

## Architecture

### G1 — Formal Specification & Proof

Candidate invariants come from domain requirements, physical constraints, economic/risk evidence, and SLOs. Candidate tau bounds come from the domain/risk model. TLA+/TLC/TLAPS may then model transitions, search for counterexamples, and prove or model-check the required properties inside the declared proof scope.

Level -1 MUST NOT interpret `proof_status=PASS` as a proof of facts outside the model or as automatic proof that the executable matches the specification.

Required outputs include:

- specification digest;
- invariant-set digest;
- assumption-set digest;
- forbidden-state-set digest;
- tau-bound digest;
- proof scope and model bounds;
- proof artifacts and toolchain identities.

### G2 — Adversarial Threat & Failure Model

The threat model defines the capability envelope and forbidden transitions against at least the classes relevant to the deployment, including poisoning, spoofing, replay, TOCTOU/latency, Byzantine sources, and privilege/capability abuse.

### G3 — Cryptographic Root of Trust

The trust layer binds source/build identities and attestation policy. Attestation establishes claims about measured software/hardware identity and freshness according to the configured policy; it does not establish that an external sensor or data source is semantically correct.

SVDC, if used by this architecture, is an **internal specification**, not an external standard. Its provenance, attestation, freshness, and anti-replay semantics belong here.

### G4 — Fail-Closed Enforcement Substrate

Default is deny. Enforcement may span multiple planes, including network controls (for example XDP), Linux security hooks/BPF-LSM, process/file/capability restrictions, and API/admission guards. A network-only latch is not sufficient for a system that can mutate state through non-network paths.

The Genesis gate verifies both that the manifest requires default-deny and that the deployment-specific external verifier can establish that the required enforcement is actually active.

### G5 — Specification-to-Implementation Refinement & Conformance Gate

G5 closes the gap:

```text
verified specification != verified implementation
```

It binds the verified specification to executable contracts, refinement mappings, source/build provenance, implementation digest, and the conformance level required for the consequence class.

Conformance levels are monotonic:

| Level | Meaning |
|---|---|
| C0 | Traceability only |
| C1 | Executable contracts and tests |
| C2 | Implementation-model refinement evidence |
| C3 | Artifact/build cryptographically bound to the verified implementation |
| C4 | Formally verified implementation and/or verified compilation where available |

A policy MAY require different minimum levels for different consequence classes. The verifier MUST reject an implementation below the required level.

## Genesis Manifest

The machine-readable schema is `genesis/genesis-manifest.schema.json`.

The manifest contains:

- version, epoch, sequence, and predecessor hash;
- specification and proof-scope identities;
- threat/capability envelope identities;
- trust-root, attestation, nonce, and revocation policy identities;
- enforcement policy and default-deny requirement;
- implementation digest, refinement mapping, executable contract, and conformance level;
- source revision, builder identity, provenance reference, material digest, and SBOM digest;
- build and validity times plus minimum accepted epoch;
- approval identities and approval-policy digest;
- detached authenticity information.

### Authenticity scope

To avoid a self-signing cycle, the signed payload is defined as:

```text
SHA-256(RFC8785(manifest with the entire `authenticity` member removed))
```

`authenticity.signed_payload_hash` carries that digest. `authenticity.signature` signs the digest according to the deployment policy. `signature_algorithm` and `signer_key_id` are inputs to policy evaluation and MUST NOT be trusted merely because the manifest names them.

The current signature scope identifier is:

```text
MANIFEST_EXCLUDING_AUTHENTICITY
```

### Anti-rollback chain

`genesis_epoch` is compared with a monotonic external minimum. `validity.minimum_accepted_epoch` may tighten that floor but cannot lower the caller's floor.

Within an epoch, `sequence > 0` requires `previous_manifest_hash`, defined as the SHA-256 digest of the RFC8785-canonicalized preceding manifest. Deployments SHOULD persist accepted epoch/sequence state in a rollback-resistant store.

### Validity and revocation

A manifest is not acceptable before `valid_from` or at/after `valid_until`. Revocation is an external fact checked through `revocation_ref` and the configured trust policy.

## Bootstrap predicate

The operational predicate is:

```text
BOOTSTRAP_ALLOWED iff
    ManifestAuthentic
and ManifestFresh
and not ManifestRevoked
and EpochNotRolledBack
and ProofScopeSatisfied
and SpecBuildConformanceSatisfied
and BuildProvenanceVerified
and TrustRootVerified
and AttestationFresh
and DefaultDenyActive
```

Any failed or unavailable required verification produces:

```text
SYSTEM_STATE = GENESIS_LOCKED
ACTION       = DENY
MUTATION     = FORBIDDEN
```

There is no permissive fallback.

## Boundary with EASL and Aegis-EGE

Level -1 establishes bootstrap legitimacy. EASL remains the domain-neutral epistemic state layer for evidence, freshness, contradictions, temporal validity, subject-state bindings, and assumption dependencies. Aegis-EGE remains a downstream policy/execution decision layer.

The sequence is:

```text
Genesis
  -> Trusted Bootstrap
  -> EASL
  -> Assumption Gate / policy resolution
  -> Aegis-EGE
  -> evidence-gated mutation
  -> observation/evidence
  -> continuous revalidation
```

A valid Genesis Manifest does not eliminate continuous evidence quality, contradiction, decay, or re-attestation requirements.

## Reference verifier

The Go package `genesis` implements the first fail-closed reference verifier. Local structural checks are combined with an `ExternalVerifier` interface for facts that cannot be proven from the JSON document alone: authenticity, trust root, attestation, revocation, build provenance, spec/build binding, runtime default-deny state, and proof requirements.

The verifier returns `BOOTSTRAP_READY` only when every required check succeeds. Otherwise it returns `GENESIS_LOCKED` plus machine-readable failure codes.
