package fixedpoint

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func ParseSource(raw []byte) (SourceSpec, error) {
	spec := SourceSpec{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		keyword := fields[0]
		rest := strings.TrimSpace(strings.TrimPrefix(line, keyword))
		switch keyword {
		case "language":
			spec.Language = rest
		case "contract":
			spec.Contract = rest
		case "toolchain":
			spec.Toolchain = rest
		case "source_read_only":
			value, err := strconv.ParseBool(rest)
			if err != nil {
				return SourceSpec{}, fmt.Errorf("line %d: source_read_only must be true or false", lineNumber)
			}
			spec.SourceReadOnly = value
		case "activity":
			declaration, err := parseDeclaration(rest)
			if err != nil {
				return SourceSpec{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			spec.Activities = append(spec.Activities, Activity{ID: declaration[0], Description: declaration[1]})
		case "stage":
			declaration, err := parseDeclaration(rest)
			if err != nil {
				return SourceSpec{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			spec.Stages = append(spec.Stages, Stage{ID: declaration[0], Description: declaration[1]})
		case "precedence":
			spec.Precedence = splitList(rest, ">")
		case "unknown_fields":
			spec.UnknownFields = splitList(rest, ",")
		case "proof":
			spec.ProofComponents = splitList(rest, ",")
		case "bootstrap":
			key, value, ok := strings.Cut(rest, "=")
			if !ok || strings.TrimSpace(key) != "compiler" || strings.TrimSpace(value) == "" {
				return SourceSpec{}, fmt.Errorf("line %d: bootstrap must declare compiler=name", lineNumber)
			}
			spec.BootstrapCompiler = strings.TrimSpace(value)
		case "bootstrap_trust":
			spec.BootstrapTrust = rest
		case "canonical_case":
			if len(fields) != 2 || fields[1] == "" {
				return SourceSpec{}, fmt.Errorf("line %d: canonical_case requires an id", lineNumber)
			}
			spec.CanonicalCases = append(spec.CanonicalCases, fields[1])
		default:
			return SourceSpec{}, fmt.Errorf("line %d: unknown semantic declaration %q", lineNumber, keyword)
		}
	}
	if err := scanner.Err(); err != nil {
		return SourceSpec{}, err
	}
	if err := ValidateSource(spec); err != nil {
		return SourceSpec{}, err
	}
	return spec, nil
}

func parseDeclaration(rest string) ([2]string, error) {
	var result [2]string
	parts := strings.SplitN(rest, "|", 2)
	left := strings.Fields(strings.TrimSpace(parts[0]))
	if len(left) != 1 || left[0] == "" {
		return result, fmt.Errorf("declaration must have one id before |")
	}
	result[0] = left[0]
	if len(parts) == 2 {
		result[1] = strings.TrimSpace(parts[1])
	}
	if result[1] == "" {
		return result, fmt.Errorf("declaration %q must have a description", result[0])
	}
	return result, nil
}

func splitList(value, separator string) []string {
	parts := strings.Split(value, separator)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func ValidateSource(spec SourceSpec) error {
	if spec.Language != "Gooo" {
		return fmt.Errorf("language must be Gooo")
	}
	if spec.Contract != "gooo-bootstrap-fixed-point/v1" {
		return fmt.Errorf("contract must be gooo-bootstrap-fixed-point/v1")
	}
	if spec.Toolchain != Toolchain {
		return fmt.Errorf("toolchain must be %s", Toolchain)
	}
	if !spec.SourceReadOnly {
		return fmt.Errorf("source_read_only must be true")
	}
	if !sameStrings(spec.Precedence, DecisionPrecedence) {
		return fmt.Errorf("precedence must be REFUTED>UNKNOWN>CLOSED")
	}
	if !sameStrings(spec.UnknownFields, RequiredUnknownFields) {
		return fmt.Errorf("unknown_fields must be stage,step,reason,unknown_class,next_operation,blocked_by")
	}
	if !sameStrings(spec.ProofComponents, RequiredProofComponents) {
		return fmt.Errorf("proof must name the four canonical fixed-point components")
	}
	if spec.BootstrapCompiler == "" || spec.BootstrapTrust != "UNPROVEN" {
		return fmt.Errorf("bootstrap boundary must name a compiler and declare UNPROVEN trust")
	}
	if len(spec.Stages) != 3 || spec.Stages[0].ID != "G0" || spec.Stages[1].ID != "G1" || spec.Stages[2].ID != "G2" {
		return fmt.Errorf("stages must be exactly G0, G1, G2 in order")
	}
	requiredActivities := []string{
		"verifier.semantic_ir",
		"verifier.provenance_graph",
		"verifier.fixed_point",
		"generator.go",
		"evaluator.behavior",
	}
	if len(spec.Activities) != len(requiredActivities) {
		return fmt.Errorf("semantic activities must declare verifier, generator, and evaluator activities")
	}
	for index, required := range requiredActivities {
		if spec.Activities[index].ID != required {
			return fmt.Errorf("activity %d must be %s", index+1, required)
		}
	}
	if len(spec.CanonicalCases) != 9 {
		return fmt.Errorf("exactly nine canonical cases are required")
	}
	seen := map[string]bool{}
	for _, caseID := range spec.CanonicalCases {
		if seen[caseID] {
			return fmt.Errorf("duplicate canonical case %q", caseID)
		}
		seen[caseID] = true
	}
	return nil
}
