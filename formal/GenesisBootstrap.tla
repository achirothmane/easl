--------------------------- MODULE GenesisBootstrap ---------------------------
EXTENDS Naturals, FiniteSets

CONSTANT RequiredChecks

VARIABLES phase, verified

vars == <<phase, verified>>

Phases == {"LOCKED", "BOOTSTRAP_READY", "RUNNING"}

GenesisValid ==
    verified = RequiredChecks

Init ==
    /\ phase = "LOCKED"
    /\ verified = {}

VerifyOne ==
    /\ phase = "LOCKED"
    /\ \E check \in RequiredChecks:
          /\ check \notin verified
          /\ verified' = verified \cup {check}
    /\ UNCHANGED phase

Invalidate ==
    /\ \E remaining \in SUBSET RequiredChecks:
          /\ remaining # RequiredChecks
          /\ verified' = remaining
    /\ phase' = "LOCKED"

Bootstrap ==
    /\ phase = "LOCKED"
    /\ GenesisValid
    /\ phase' = "BOOTSTRAP_READY"
    /\ UNCHANGED verified

Start ==
    /\ phase = "BOOTSTRAP_READY"
    /\ GenesisValid
    /\ phase' = "RUNNING"
    /\ UNCHANGED verified

Next ==
    VerifyOne \/ Invalidate \/ Bootstrap \/ Start

Spec ==
    Init /\ [][Next]_vars

TypeOK ==
    /\ phase \in Phases
    /\ verified \subseteq RequiredChecks

NoBootstrapWithoutValidGenesis ==
    phase # "LOCKED" => GenesisValid

NoMutationWithoutValidGenesis ==
    phase = "RUNNING" => GenesisValid

=============================================================================
