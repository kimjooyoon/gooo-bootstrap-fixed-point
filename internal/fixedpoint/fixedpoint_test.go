package fixedpoint

import "testing"

func TestUnknownRequiresAllSixFields(t *testing.T) {
	unknown := unknownClaim("EVIDENCE", "OBSERVE", "not available", "MISSING", "COLLECT", []string{"fixture"})
	if !unknown.Valid() {
		t.Fatal("complete UNKNOWN tuple must be valid")
	}
	unknown.BlockedBy = nil
	if unknown.Valid() {
		t.Fatal("UNKNOWN without blocked_by must not be valid")
	}
}

func TestUnknownTopLevelDecisionFailsClosed(t *testing.T) {
	result := EvaluateCase(CaseFixture{
		ID: "unknown-decision", DeclaredDecision: "MAYBE", ExpectedDecision: DecisionRefuted,
	})
	if result.Decision != DecisionRefuted || result.Reason != "UNKNOWN_TOP_LEVEL_DECISION_FAIL_CLOSED" {
		t.Fatalf("unexpected fail-closed result: %#v", result)
	}
}

func TestPrecedenceIsRefutedUnknownClosed(t *testing.T) {
	if got := resolveDecision(DecisionClosed, DecisionUnknown, DecisionRefuted); got != DecisionRefuted {
		t.Fatalf("precedence must resolve to REFUTED, got %s", got)
	}
}

func TestContradictionDominatesUnknown(t *testing.T) {
	result := EvaluateCase(CaseFixture{
		ID: "precedence", DeclaredDecision: DecisionUnknown, ExpectedDecision: DecisionRefuted,
		Contradiction: true, Unknown: unknownClaim("DECISION", "RESOLVE", "evidence conflict", "MIXED", "REPAIR", []string{"contradiction"}),
	})
	if result.Decision != DecisionRefuted || !result.ExpectedMatch {
		t.Fatalf("contradiction must dominate UNKNOWN: %#v", result)
	}
}

func TestSemanticIRIncludesDeclaredStagesAndActivities(t *testing.T) {
	spec := SourceSpec{
		Language: "Gooo", Contract: "gooo-bootstrap-fixed-point/v1", Toolchain: Toolchain,
		SourceReadOnly: true,
		Activities: []Activity{{ID: "verifier.semantic_ir"}, {ID: "verifier.provenance_graph"}, {ID: "verifier.fixed_point"}, {ID: "generator.go"}, {ID: "evaluator.behavior"}},
		Stages: []Stage{{ID: "G0"}, {ID: "G1"}, {ID: "G2"}},
		Precedence: DecisionPrecedence, UnknownFields: RequiredUnknownFields, ProofComponents: RequiredProofComponents,
		BootstrapCompiler: "seed-compiler", BootstrapTrust: "UNPROVEN", CanonicalCases: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"},
	}
	ir := BuildSemanticIR(spec, "sha256:source")
	if len(ir.Activities) != 5 || len(ir.Stages) != 3 || !ir.SourceReadOnly {
		t.Fatalf("semantic declarations were not preserved: %#v", ir)
	}
}

func TestGenerationIsStableAcrossG1AndG2(t *testing.T) {
	spec := SourceSpec{
		Language: "Gooo", Contract: "gooo-bootstrap-fixed-point/v1", Toolchain: Toolchain,
		SourceReadOnly: true, Precedence: DecisionPrecedence, UnknownFields: RequiredUnknownFields,
		ProofComponents: RequiredProofComponents, BootstrapCompiler: "seed-compiler", BootstrapTrust: "UNPROVEN",
	}
	g1, err := GenerateStage(spec, "sha256:source", "G1")
	if err != nil {
		t.Fatal(err)
	}
	g2, err := GenerateStage(spec, "sha256:source", "G2")
	if err != nil {
		t.Fatal(err)
	}
	if g1.GeneratedGo != g2.GeneratedGo || g1.BehaviorOutput != g2.BehaviorOutput {
		t.Fatal("G1 and G2 generated behavior must be exact matches")
	}
	left := mustCanonical(g1.ProvenanceGraph)
	right := mustCanonical(g2.ProvenanceGraph)
	if string(left) != string(right) {
		t.Fatal("G1 and G2 provenance graphs must be exact matches")
	}
}
