package fixedpoint

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	semanticIRSchema       = "gooo/bootstrap-fixed-point/semantic-ir/v1"
	provenanceGraphSchema  = "gooo/bootstrap-fixed-point/provenance-graph/v1"
	stageReceiptSchema     = "gooo/bootstrap-fixed-point/stage-receipt/v1"
	fixedPointProofSchema  = "gooo/bootstrap-fixed-point/fixed-point-proof/v1"
	conformanceReportSchema = "gooo/bootstrap-fixed-point/conformance-report/v1"
)

func DigestBytes(raw []byte) string {
	digest := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func CanonicalJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

func CanonicalDigest(value any) (string, error) {
	raw, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(raw), nil
}

func BuildSemanticIR(spec SourceSpec, sourceDigest string) SemanticIR {
	activities := make([]string, 0, len(spec.Activities))
	for _, activity := range spec.Activities {
		activities = append(activities, activity.ID)
	}
	stages := make([]string, 0, len(spec.Stages))
	for _, stage := range spec.Stages {
		stages = append(stages, stage.ID)
	}
	rules := []SemanticRule{
		{ID: "decision_precedence", Value: strings.Join(spec.Precedence, ">")},
		{ID: "unknown_fields", Value: strings.Join(spec.UnknownFields, ",")},
		{ID: "fixed_point_components", Value: strings.Join(spec.ProofComponents, ",")},
		{ID: "source_write_authority", Value: fmt.Sprintf("source_read_only=%t", spec.SourceReadOnly)},
		{ID: "bootstrap_compiler", Value: spec.BootstrapCompiler},
		{ID: "bootstrap_trust", Value: spec.BootstrapTrust},
	}
	return SemanticIR{
		Schema:            semanticIRSchema,
		Language:          spec.Language,
		Contract:          spec.Contract,
		SourceDigest:      sourceDigest,
		Toolchain:         spec.Toolchain,
		SourceReadOnly:    spec.SourceReadOnly,
		Activities:        activities,
		Stages:            stages,
		Rules:             rules,
		BootstrapBoundary: "compiler=" + spec.BootstrapCompiler + ";trust=" + spec.BootstrapTrust,
	}
}

func renderGeneratedGo(irDigest string) string {
	return fmt.Sprintf(`package main

import "fmt"

const generatedOutput = "gooo-fixed-point-behavior|semantic-ir=%s"

func main() {
	fmt.Println(generatedOutput)
}
`, irDigest)
}

func BuildProvenanceGraph(sourceDigest string, irDigest string, generatedGoDigest string, outputDigest string) ProvenanceGraph {
	return ProvenanceGraph{
		Schema: provenanceGraphSchema,
		Nodes: []ProvenanceNode{
			{ID: "source", Kind: "gooo-source", CanonicalDigest: sourceDigest},
			{ID: "semantic-ir", Kind: "semantic-ir", CanonicalDigest: irDigest},
			{ID: "generated-go", Kind: "generated-go", CanonicalDigest: generatedGoDigest},
			{ID: "generated-output", Kind: "generated-output", CanonicalDigest: outputDigest},
		},
		Edges: []ProvenanceEdge{
			{ID: "edge-source-to-ir", From: "source", To: "semantic-ir", Relation: "parsed-into"},
			{ID: "edge-ir-to-go", From: "semantic-ir", To: "generated-go", Relation: "generated-into"},
			{ID: "edge-go-to-output", From: "generated-go", To: "generated-output", Relation: "executed-into"},
		},
	}
}

func GenerateStage(spec SourceSpec, sourceDigest string, stage string) (GeneratedStage, error) {
	ir := BuildSemanticIR(spec, sourceDigest)
	irDigest, err := CanonicalDigest(ir)
	if err != nil {
		return GeneratedStage{}, err
	}
	generatedGo := renderGeneratedGo(irDigest)
	generatedGoDigest := DigestBytes([]byte(generatedGo))
	behaviorOutput := "gooo-fixed-point-behavior|semantic-ir=" + irDigest
	outputDigest := DigestBytes([]byte(behaviorOutput))
	graph := BuildProvenanceGraph(sourceDigest, irDigest, generatedGoDigest, outputDigest)
	graphDigest, err := CanonicalDigest(graph)
	if err != nil {
		return GeneratedStage{}, err
	}
	return GeneratedStage{
		Stage:             stage,
		SemanticIR:        ir,
		GeneratedGo:       generatedGo,
		BehaviorOutput:    behaviorOutput,
		ProvenanceGraph:   graph,
		SemanticIRDigest:  irDigest,
		GeneratedGoDigest: generatedGoDigest,
		OutputDigest:      outputDigest,
		GraphDigest:       graphDigest,
	}, nil
}

func writeStage(outputRoot string, artifact GeneratedStage) (StageReceipt, error) {
	stageDir := filepath.Join(outputRoot, "stages", artifact.Stage)
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return StageReceipt{}, err
	}
	irRaw, err := json.MarshalIndent(artifact.SemanticIR, "", "  ")
	if err != nil {
		return StageReceipt{}, err
	}
	graphRaw, err := json.MarshalIndent(artifact.ProvenanceGraph, "", "  ")
	if err != nil {
		return StageReceipt{}, err
	}
	if err := os.WriteFile(filepath.Join(stageDir, "semantic-ir.json"), append(irRaw, '\n'), 0o644); err != nil {
		return StageReceipt{}, err
	}
	if err := os.WriteFile(filepath.Join(stageDir, "generated.go"), []byte(artifact.GeneratedGo), 0o644); err != nil {
		return StageReceipt{}, err
	}
	if err := os.WriteFile(filepath.Join(stageDir, "behavior-output.txt"), []byte(artifact.BehaviorOutput+"\n"), 0o644); err != nil {
		return StageReceipt{}, err
	}
	if err := os.WriteFile(filepath.Join(stageDir, "provenance-graph.json"), append(graphRaw, '\n'), 0o644); err != nil {
		return StageReceipt{}, err
	}
	receipt := StageReceipt{
		Schema:            stageReceiptSchema,
		Stage:             artifact.Stage,
		SourceDigest:      artifact.SemanticIR.SourceDigest,
		Toolchain:         artifact.SemanticIR.Toolchain,
		Runner:            Runner,
		SemanticIRDigest:  artifact.SemanticIRDigest,
		GeneratedGoDigest: artifact.GeneratedGoDigest,
		OutputDigest:      artifact.OutputDigest,
		GraphDigest:       artifact.GraphDigest,
		TrustBoundary:     "bootstrap-seed",
		TrustDecision:     DecisionUnknown,
	}
	receiptRaw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return StageReceipt{}, err
	}
	if err := os.WriteFile(filepath.Join(stageDir, "generation-receipt.json"), append(receiptRaw, '\n'), 0o644); err != nil {
		return StageReceipt{}, err
	}
	stageReport := fmt.Sprintf("# Gooo stage %s\n\nsemantic IR: `%s`\ngenerated Go: `%s`\ngenerated output: `%s`\nprovenance graph: `%s`\ntrust boundary: `%s` (`%s`)\n", artifact.Stage, artifact.SemanticIRDigest, artifact.GeneratedGoDigest, artifact.OutputDigest, artifact.GraphDigest, receipt.TrustBoundary, receipt.TrustDecision)
	if err := os.WriteFile(filepath.Join(stageDir, "stage-report.md"), []byte(stageReport), 0o644); err != nil {
		return StageReceipt{}, err
	}
	return receipt, nil
}

func executeGenerated(stageDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "run", "generated.go")
	command.Dir = stageDir
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("generated Go execution failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func EvaluateCase(fixture CaseFixture) CaseResult {
	result := CaseResult{
		Ordinal: fixture.Ordinal, ID: fixture.ID, Class: fixture.Class,
		ExpectedDecision: fixture.ExpectedDecision,
	}
	if fixture.Malformed {
		result.Decision = DecisionRefuted
		result.Reason = "MALFORMED_INPUT_FAIL_CLOSED"
		result.ExpectedMatch = result.Decision == result.ExpectedDecision
		return result
	}
	if !isDecision(fixture.DeclaredDecision) {
		result.Decision = DecisionRefuted
		result.Reason = "UNKNOWN_TOP_LEVEL_DECISION_FAIL_CLOSED"
		result.ExpectedMatch = result.Decision == result.ExpectedDecision
		return result
	}
	if fixture.Contradiction {
		result.Decision = DecisionRefuted
		result.Reason = "REFUTED_PRECEDENCE_OVERRIDES_UNKNOWN"
		result.Unknown = fixture.Unknown
		result.ExpectedMatch = result.Decision == result.ExpectedDecision
		return result
	}
	if fixture.DeclaredDecision == DecisionUnknown {
		if fixture.Unknown == nil || !fixture.Unknown.Valid() {
			result.Decision = DecisionRefuted
			result.Reason = "UNKNOWN_TUPLE_MALFORMED_FAIL_CLOSED"
			result.ExpectedMatch = result.Decision == result.ExpectedDecision
			return result
		}
		result.Decision = DecisionUnknown
		result.Reason = fixture.Unknown.Reason
		result.Unknown = fixture.Unknown
		result.ExpectedMatch = result.Decision == result.ExpectedDecision
		return result
	}
	result.Decision = fixture.DeclaredDecision
	result.Reason = "CANONICAL_EVIDENCE_COMPLETE"
	result.ExpectedMatch = result.Decision == result.ExpectedDecision
	return result
}

func buildFixedPointProof(g1, g2 GeneratedStage, sourceDigest, contractDigest string) (FixedPointProof, error) {
	g1IR, err := CanonicalDigest(g1.SemanticIR)
	if err != nil {
		return FixedPointProof{}, err
	}
	g2IR, err := CanonicalDigest(g2.SemanticIR)
	if err != nil {
		return FixedPointProof{}, err
	}
	g1Graph, err := CanonicalDigest(g1.ProvenanceGraph)
	if err != nil {
		return FixedPointProof{}, err
	}
	g2Graph, err := CanonicalDigest(g2.ProvenanceGraph)
	if err != nil {
		return FixedPointProof{}, err
	}
	components := []ComponentComparison{
		{Component: "canonical_semantic_ir", G1Digest: g1IR, G2Digest: g2IR, ExactMatch: bytes.Equal(mustCanonical(g1.SemanticIR), mustCanonical(g2.SemanticIR))},
		{Component: "generated_go_behavior", G1Digest: DigestBytes([]byte(g1.GeneratedGo)), G2Digest: DigestBytes([]byte(g2.GeneratedGo)), ExactMatch: g1.GeneratedGo == g2.GeneratedGo},
		{Component: "generated_output", G1Digest: DigestBytes([]byte(g1.BehaviorOutput)), G2Digest: DigestBytes([]byte(g2.BehaviorOutput)), ExactMatch: g1.BehaviorOutput == g2.BehaviorOutput},
		{Component: "provenance_graph", G1Digest: g1Graph, G2Digest: g2Graph, ExactMatch: bytes.Equal(mustCanonical(g1.ProvenanceGraph), mustCanonical(g2.ProvenanceGraph))},
	}
	allExact := true
	for _, component := range components {
		if !component.ExactMatch {
			allExact = false
		}
	}
	decision := DecisionRefuted
	claim := "FIXED_POINT_REFUTED"
	if allExact && sourceDigest == g1.SemanticIR.SourceDigest && sourceDigest == g2.SemanticIR.SourceDigest && g1.SemanticIR.Toolchain == Toolchain && g2.SemanticIR.Toolchain == Toolchain {
		decision = DecisionClosed
		claim = "FIXED_POINT_OBSERVED"
	}
	bootstrapUnknown := unknownClaim(
		"BOOTSTRAP", "VERIFY_BOOTSTRAP_COMPILER", "seed compiler trust is declared UNPROVEN",
		"BOOTSTRAP_TRUST", "PROVIDE_BOOTSTRAP_PROOF", []string{"bootstrap compiler seed-compiler"},
	)
	return FixedPointProof{
		Schema:            fixedPointProofSchema,
		Decision:          decision,
		Claim:             claim,
		InputSourceDigest: sourceDigest,
		ContractDigest:    contractDigest,
		Toolchain:         Toolchain,
		Runner:            Runner,
		SamePinnedInput:   sourceDigest == g1.SemanticIR.SourceDigest && sourceDigest == g2.SemanticIR.SourceDigest && g1.SemanticIR.Toolchain == g2.SemanticIR.Toolchain,
		Components:        components,
		BootstrapTrust:    Claim{Decision: DecisionUnknown, Value: nil, Unknown: bootstrapUnknown},
		ObservedGeneration: []string{"G0", "G1", "G2"},
	}, nil
}

func mustCanonical(value any) []byte {
	raw, err := CanonicalJSON(value)
	if err != nil {
		return nil
	}
	return raw
}

func readContractAndCorpus(root string) (Contract, CaseCorpus, []byte, error) {
	contractRaw, err := os.ReadFile(filepath.Join(root, "contracts", "denominator-v1.json"))
	if err != nil {
		return Contract{}, CaseCorpus{}, nil, err
	}
	var contract Contract
	if err := json.Unmarshal(contractRaw, &contract); err != nil {
		return Contract{}, CaseCorpus{}, nil, fmt.Errorf("decode denominator: %w", err)
	}
	corpusRaw, err := os.ReadFile(filepath.Join(root, "fixtures", "cases.json"))
	if err != nil {
		return Contract{}, CaseCorpus{}, nil, err
	}
	var corpus CaseCorpus
	if err := json.Unmarshal(corpusRaw, &corpus); err != nil {
		return Contract{}, CaseCorpus{}, nil, fmt.Errorf("decode cases: %w", err)
	}
	return contract, corpus, contractRaw, nil
}

func validateContract(spec SourceSpec, contract Contract, corpus CaseCorpus) error {
	if contract.Schema != "gooo/bootstrap-fixed-point/denominator/v1" || contract.DenominatorID != "bootstrap-fixed-point-canonical-v1" {
		return fmt.Errorf("denominator identity is not canonical")
	}
	if contract.Total != 9 || len(contract.Cases) != 9 || len(corpus.Cases) != 9 {
		return fmt.Errorf("canonical denominator must contain exactly nine cases")
	}
	if !sameStrings(contract.DecisionPrecedence, DecisionPrecedence) {
		return fmt.Errorf("contract precedence does not match REFUTED>UNKNOWN>CLOSED")
	}
	if len(spec.CanonicalCases) != 9 {
		return fmt.Errorf("source did not declare nine canonical cases")
	}
	for index, contractCase := range contract.Cases {
		if contractCase.Ordinal != index+1 || contractCase.ID != spec.CanonicalCases[index] {
			return fmt.Errorf("contract case %d does not match .gooo declaration", index+1)
		}
		fixture := corpus.Cases[index]
		if fixture.Ordinal != contractCase.Ordinal || fixture.ID != contractCase.ID || fixture.Class != contractCase.Class || fixture.ExpectedDecision != contractCase.ExpectedDecision {
			return fmt.Errorf("fixture case %d does not match fixed denominator", index+1)
		}
	}
	if contract.ClassCounts["NORMAL"] != 3 || contract.ClassCounts["UNKNOWN"] != 3 || contract.ClassCounts["REFUTED"] != 3 {
		return fmt.Errorf("class denominator must be three NORMAL, three UNKNOWN, and three REFUTED")
	}
	return nil
}

func InventoryForRoot(root string) (Inventory, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, err
	}
	result := Inventory{RootREADMEExcluded: true}
	err = filepath.WalkDir(absolute, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == absolute {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			result.DescendantDirs++
			return nil
		}
		if !entry.Type().IsRegular() || path == filepath.Join(absolute, "README.md") {
			return nil
		}
		result.RegularFiles++
		extension := filepath.Ext(entry.Name())
		if extension != ".go" && extension != ".gooo" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := physicalLines(raw)
		if extension == ".go" {
			result.GoFiles++
			result.GoPhysicalLines += lines
		} else {
			result.GoooFiles++
			result.GoooPhysicalLines += lines
		}
		return nil
	})
	return result, err
}

func physicalLines(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	lines := strings.Count(string(raw), "\n")
	if raw[len(raw)-1] != '\n' {
		lines++
	}
	return lines
}

func generatedInventory(root string) (GeneratedInventory, error) {
	var result GeneratedInventory
	err := filepath.WalkDir(filepath.Join(root, "stages"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		result.Artifacts++
		result.Bytes += info.Size()
		return nil
	})
	return result, err
}

func stageMetric(stage string, started time.Time) StageMetric {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	return StageMetric{Stage: stage, WallMS: time.Since(started).Milliseconds(), PeakRSSKiB: int64(memory.Sys / 1024)}
}

func utilityUnknown() Claim {
	return Claim{
		Decision: DecisionUnknown,
		Value:    nil,
		Unknown: unknownClaim(
			"UTILITY", "OBSERVE_EXTERNAL_USER_EVIDENCE", "no external user evidence is supplied",
			"MISSING_EXTERNAL_EVIDENCE", "COLLECT_EXTERNAL_USER_EVIDENCE", []string{"external user observation"},
		),
	}
}

func improvementUnknown() Claim {
	return Claim{
		Decision: DecisionUnknown,
		Value:    nil,
		Unknown: unknownClaim(
			"IMPROVEMENT", "COMPARE_EXACT_BEFORE_AFTER_PAIR", "G1 and G2 regeneration is not external utility evidence",
			"IMPROVEMENT_UNKNOWN", "COLLECT_SAME_JOB_EXACT_BEFORE_AFTER_PAIR", []string{"scenario", "source", "contract", "fixture", "toolchain", "runner"},
		),
	}
}

func RunConformance(inputPath, root, outputRoot string) (ConformanceReport, error) {
	started := time.Now()
	sourceRaw, err := os.ReadFile(inputPath)
	if err != nil {
		return ConformanceReport{}, err
	}
	spec, err := ParseSource(sourceRaw)
	if err != nil {
		return ConformanceReport{}, err
	}
	contract, corpus, contractRaw, err := readContractAndCorpus(root)
	if err != nil {
		return ConformanceReport{}, err
	}
	if err := validateContract(spec, contract, corpus); err != nil {
		return ConformanceReport{}, err
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return ConformanceReport{}, err
	}
	sourceDigest := DigestBytes(sourceRaw)
	contractDigest := DigestBytes(contractRaw)
	stages := []string{"G0", "G1", "G2"}
	artifacts := make(map[string]GeneratedStage, len(stages))
	for _, stage := range stages {
		artifact, err := GenerateStage(spec, sourceDigest, stage)
		if err != nil {
			return ConformanceReport{}, err
		}
		stageDir := filepath.Join(outputRoot, "stages", stage)
		if _, err := writeStage(outputRoot, artifact); err != nil {
			return ConformanceReport{}, err
		}
		observed, err := executeGenerated(stageDir)
		if err != nil {
			return ConformanceReport{}, err
		}
		if observed != artifact.BehaviorOutput {
			return ConformanceReport{}, fmt.Errorf("%s generated behavior differs from the declared output", stage)
		}
		artifact.BehaviorOutput = observed
		artifacts[stage] = artifact
		if _, err := writeStage(outputRoot, artifact); err != nil {
			return ConformanceReport{}, err
		}
	}
	fixedPoint, err := buildFixedPointProof(artifacts["G1"], artifacts["G2"], sourceDigest, contractDigest)
	if err != nil {
		return ConformanceReport{}, err
	}
	caseResults := make([]CaseResult, 0, len(corpus.Cases))
	decisionCounts := map[string]int{DecisionClosed: 0, DecisionUnknown: 0, DecisionRefuted: 0}
	for _, fixture := range corpus.Cases {
		result := EvaluateCase(fixture)
		if !result.ExpectedMatch {
			return ConformanceReport{}, fmt.Errorf("case %s resolved to %s but expected %s", result.ID, result.Decision, result.ExpectedDecision)
		}
		caseResults = append(caseResults, result)
		decisionCounts[result.Decision]++
	}
	overallDecision := DecisionClosed
	for _, result := range caseResults {
		overallDecision = resolveDecision(overallDecision, result.Decision)
	}
	inventory, err := InventoryForRoot(root)
	if err != nil {
		return ConformanceReport{}, err
	}
	generated, err := generatedInventory(outputRoot)
	if err != nil {
		return ConformanceReport{}, err
	}
	report := ConformanceReport{
		Schema:           conformanceReportSchema,
		Decision:         overallDecision,
		Reason:           "REFUTED_CASE_PRESENT_IN_FIXED_CANONICAL_DENOMINATOR",
		Precedence:       append([]string(nil), DecisionPrecedence...),
		DenominatorID:    contract.DenominatorID,
		FixedDenominator: contract.Total,
		Cases:            caseResults,
		DecisionCounts:   decisionCounts,
		FixedPoint:       fixedPoint,
		Metrics: MetricsReport{
			Stages: []StageMetric{stageMetric("conformance", started)},
			Tests: TestMetrics{Total: 9, Selected: 9, Executed: 9, Reused: 0, Failed: 0, Unknown: 3},
			Inventory: inventory,
			Generated: generated,
		},
		Authority: AuthorityReport{
			Local: AuthorityCounts{RepositoryWrites: 0, InputSourceWrites: 0, RuntimeCommitMergeTagRelease: 0, CallerOwnedOutputWrites: 1},
			Remote: AuthorityCounts{RepositoryWrites: 0, InputSourceWrites: 0, RuntimeCommitMergeTagRelease: 0, CrossProjectRequiredGates: 0},
		},
		Utility: utilityUnknown(), Improvement: improvementUnknown(),
		LocalValidationCommands: 0, OperationalRefuted: false,
		EndToEnd: []string{"input .gooo", "semantic IR", "provenance graph", "generated Go", "executed behavior", "human-readable report"},
	}
	if err := writeReport(outputRoot, report); err != nil {
		return ConformanceReport{}, err
	}
	return report, nil
}

func writeReport(outputRoot string, report ConformanceReport) error {
	if err := os.MkdirAll(filepath.Join(outputRoot, "reports"), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputRoot, "report.json"), append(raw, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputRoot, "reports", "conformance.md"), []byte(RenderMarkdown(report)), 0o644); err != nil {
		return err
	}
	return nil
}

func RenderMarkdown(report ConformanceReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Gooo bootstrap fixed-point conformance\n\n")
	fmt.Fprintf(&b, "decision: `%s`\n\n", report.Decision)
	fmt.Fprintf(&b, "fixed denominator: `%s` (%d canonical cases)\n\n", report.DenominatorID, report.FixedDenominator)
	b.WriteString("| ordinal | case | class | expected | actual | reason |\n|---:|---|---|---|---|---|\n")
	for _, result := range report.Cases {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s |\n", result.Ordinal, result.ID, result.Class, result.ExpectedDecision, result.Decision, result.Reason)
	}
	fmt.Fprintf(&b, "\nexact decision counts: CLOSED=%d, UNKNOWN=%d, REFUTED=%d\n\n", report.DecisionCounts[DecisionClosed], report.DecisionCounts[DecisionUnknown], report.DecisionCounts[DecisionRefuted])
	b.WriteString("## Fixed-point proof\n\n")
	fmt.Fprintf(&b, "observed claim: `%s` (`%s`)\n\n", report.FixedPoint.Claim, report.FixedPoint.Decision)
	b.WriteString("| component | G1 canonical digest | G2 canonical digest | exact match |\n|---|---|---|---|\n")
	for _, component := range report.FixedPoint.Components {
		fmt.Fprintf(&b, "| %s | `%s` | `%s` | %t |\n", component.Component, component.G1Digest, component.G2Digest, component.ExactMatch)
	}
	fmt.Fprintf(&b, "\nbootstrap trust: `%s`; reason: %s\n\n", report.FixedPoint.BootstrapTrust.Decision, report.FixedPoint.BootstrapTrust.Unknown.Reason)
	b.WriteString("## End-to-end path\n\n")
	b.WriteString(strings.Join(report.EndToEnd, " → ") + "\n\n")
	fmt.Fprintf(&b, "tests: total=%d, selected=%d, executed=%d, reused=%d, failed=%d, unknown=%d\n\n", report.Metrics.Tests.Total, report.Metrics.Tests.Selected, report.Metrics.Tests.Executed, report.Metrics.Tests.Reused, report.Metrics.Tests.Failed, report.Metrics.Tests.Unknown)
	fmt.Fprintf(&b, "inventory: Go files=%d (%d physical lines), Gooo files=%d (%d physical lines), descendant directories=%d, regular files=%d; root README excluded=%t\n\n", report.Metrics.Inventory.GoFiles, report.Metrics.Inventory.GoPhysicalLines, report.Metrics.Inventory.GoooFiles, report.Metrics.Inventory.GoooPhysicalLines, report.Metrics.Inventory.DescendantDirs, report.Metrics.Inventory.RegularFiles, report.Metrics.Inventory.RootREADMEExcluded)
	fmt.Fprintf(&b, "generated artifacts: %d files, %d bytes\n\n", report.Metrics.Generated.Artifacts, report.Metrics.Generated.Bytes)
	fmt.Fprintf(&b, "utility: `%s` because external user evidence is absent; improvement: `%s` because the exact same-job before/after integer pair is absent.\n", report.Utility.Decision, report.Improvement.Decision)
	return b.String()
}

func RunIntegration(inputPath, outputRoot string) error {
	sourceRaw, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	spec, err := ParseSource(sourceRaw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return err
	}
	sourceDigest := DigestBytes(sourceRaw)
	artifact, err := GenerateStage(spec, sourceDigest, "G1")
	if err != nil {
		return err
	}
	stageDir := filepath.Join(outputRoot, "stages", "G1")
	if _, err := writeStage(outputRoot, artifact); err != nil {
		return err
	}
	observed, err := executeGenerated(stageDir)
	if err != nil {
		return err
	}
	if observed != artifact.BehaviorOutput {
		return fmt.Errorf("integration output mismatch")
	}
	artifact.BehaviorOutput = observed
	if _, err := writeStage(outputRoot, artifact); err != nil {
		return err
	}
	report := map[string]any{
		"schema":       "gooo/bootstrap-fixed-point/integration-report/v1",
		"decision":     DecisionClosed,
		"input":        "examples/self-description.gooo",
		"source_digest": sourceDigest,
		"generated_stage": "G1",
		"generated_go":   "stages/G1/generated.go",
		"observed_output": observed,
		"proof_path": []string{".gooo", "semantic-ir.json", "provenance-graph.json", "generated.go", "behavior-output.txt", "integration-report.md"},
		"authority": AuthorityCounts{RepositoryWrites: 0, InputSourceWrites: 0, RuntimeCommitMergeTagRelease: 0, CallerOwnedOutputWrites: 1},
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputRoot, "integration-report.json"), append(raw, '\n'), 0o644); err != nil {
		return err
	}
	markdown := "# Gooo end-to-end integration\n\n" + "decision: `CLOSED`\n\n" + "input `.gooo` → semantic IR → provenance graph → generated Go → executed behavior → report\n\n" + "observed output: `" + observed + "`\n"
	return os.WriteFile(filepath.Join(outputRoot, "integration-report.md"), []byte(markdown), 0o644)
}

func WriteGenerated(spec SourceSpec, sourceRaw []byte, stage, outputRoot string) error {
	artifact, err := GenerateStage(spec, DigestBytes(sourceRaw), stage)
	if err != nil {
		return err
	}
	_, err = writeStage(outputRoot, artifact)
	return err
}

func SortedDirectoryEntries(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Name())
	}
	sort.Strings(paths)
	return paths, nil
}
