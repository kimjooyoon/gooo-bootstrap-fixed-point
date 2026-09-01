package fixedpoint

import "strings"

const (
	DecisionClosed  = "CLOSED"
	DecisionUnknown = "UNKNOWN"
	DecisionRefuted = "REFUTED"
	Toolchain       = "go1.27.0"
	Runner          = "github-actions/ubuntu-latest"
)

var DecisionPrecedence = []string{DecisionRefuted, DecisionUnknown, DecisionClosed}

var RequiredUnknownFields = []string{
	"stage",
	"step",
	"reason",
	"unknown_class",
	"next_operation",
	"blocked_by",
}

var RequiredProofComponents = []string{
	"canonical_semantic_ir",
	"generated_go_behavior",
	"generated_output",
	"provenance_graph",
}

type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

func (u Unknown) Valid() bool {
	return u.Stage != "" && u.Step != "" && u.Reason != "" &&
		u.UnknownClass != "" && u.NextOperation != "" && u.BlockedBy != nil
}

func unknownClaim(stage, step, reason, class, next string, blockedBy []string) *Unknown {
	return &Unknown{
		Stage: stage, Step: step, Reason: reason, UnknownClass: class,
		NextOperation: next, BlockedBy: append([]string(nil), blockedBy...),
	}
}

type Activity struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type Stage struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type SourceSpec struct {
	Language         string     `json:"language"`
	Contract         string     `json:"contract"`
	Toolchain        string     `json:"toolchain"`
	SourceReadOnly   bool       `json:"source_read_only"`
	Activities       []Activity `json:"activities"`
	Stages           []Stage    `json:"stages"`
	Precedence       []string   `json:"precedence"`
	UnknownFields    []string   `json:"unknown_fields"`
	ProofComponents  []string   `json:"proof_components"`
	BootstrapCompiler string    `json:"bootstrap_compiler"`
	BootstrapTrust   string     `json:"bootstrap_trust"`
	CanonicalCases   []string   `json:"canonical_cases"`
}

type Contract struct {
	Schema            string            `json:"schema"`
	DenominatorID     string            `json:"denominator_id"`
	Total             int               `json:"total"`
	DecisionPrecedence []string         `json:"decision_precedence"`
	ClassCounts       map[string]int    `json:"class_counts"`
	Cases             []ContractCase    `json:"cases"`
}

type ContractCase struct {
	Ordinal         int    `json:"ordinal"`
	ID              string `json:"id"`
	Class           string `json:"class"`
	ExpectedDecision string `json:"expected_decision"`
}

type CaseFixture struct {
	Ordinal          int      `json:"ordinal"`
	ID               string   `json:"id"`
	Class            string   `json:"class"`
	Description      string   `json:"description"`
	DeclaredDecision string   `json:"declared_decision"`
	ExpectedDecision string   `json:"expected_decision"`
	Malformed        bool     `json:"malformed"`
	Contradiction    bool     `json:"contradiction"`
	Unknown          *Unknown `json:"unknown,omitempty"`
}

type CaseCorpus struct {
	Schema string        `json:"schema"`
	Cases  []CaseFixture `json:"cases"`
}

type SemanticRule struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type SemanticIR struct {
	Schema          string         `json:"schema"`
	Language        string         `json:"language"`
	Contract        string         `json:"contract"`
	SourceDigest    string         `json:"source_digest"`
	Toolchain       string         `json:"toolchain"`
	SourceReadOnly  bool           `json:"source_read_only"`
	Activities      []string       `json:"activities"`
	Stages          []string       `json:"stages"`
	Rules           []SemanticRule `json:"rules"`
	BootstrapBoundary string       `json:"bootstrap_boundary"`
}

type ProvenanceNode struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	CanonicalDigest string `json:"canonical_digest"`
}

type ProvenanceEdge struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

type ProvenanceGraph struct {
	Schema       string           `json:"schema"`
	Nodes        []ProvenanceNode `json:"nodes"`
	Edges        []ProvenanceEdge `json:"edges"`
}

type GeneratedStage struct {
	Stage             string          `json:"stage"`
	SemanticIR        SemanticIR      `json:"semantic_ir"`
	GeneratedGo       string          `json:"generated_go"`
	BehaviorOutput    string          `json:"behavior_output"`
	ProvenanceGraph   ProvenanceGraph `json:"provenance_graph"`
	SemanticIRDigest  string          `json:"semantic_ir_digest"`
	GeneratedGoDigest string          `json:"generated_go_digest"`
	OutputDigest      string          `json:"output_digest"`
	GraphDigest       string          `json:"graph_digest"`
}

type StageReceipt struct {
	Schema            string `json:"schema"`
	Stage             string `json:"stage"`
	SourceDigest      string `json:"source_digest"`
	Toolchain         string `json:"toolchain"`
	Runner            string `json:"runner"`
	SemanticIRDigest  string `json:"semantic_ir_digest"`
	GeneratedGoDigest string `json:"generated_go_digest"`
	OutputDigest      string `json:"output_digest"`
	GraphDigest       string `json:"graph_digest"`
	TrustBoundary     string `json:"trust_boundary"`
	TrustDecision     string `json:"trust_decision"`
}

type ComponentComparison struct {
	Component string `json:"component"`
	G1Digest  string `json:"g1_digest"`
	G2Digest  string `json:"g2_digest"`
	ExactMatch bool   `json:"exact_match"`
}

type Claim struct {
	Decision string   `json:"decision"`
	Value    *int     `json:"value"`
	Unknown  *Unknown `json:"unknown,omitempty"`
}

type FixedPointProof struct {
	Schema             string               `json:"schema"`
	Decision           string               `json:"decision"`
	Claim              string               `json:"claim"`
	InputSourceDigest  string               `json:"input_source_digest"`
	ContractDigest     string               `json:"contract_digest"`
	Toolchain          string               `json:"toolchain"`
	Runner             string               `json:"runner"`
	SamePinnedInput    bool                 `json:"same_pinned_input"`
	Components         []ComponentComparison `json:"components"`
	BootstrapTrust     Claim                `json:"bootstrap_trust"`
	ObservedGeneration []string             `json:"observed_generation"`
}

type CaseResult struct {
	Ordinal          int      `json:"ordinal"`
	ID               string   `json:"id"`
	Class            string   `json:"class"`
	ExpectedDecision string   `json:"expected_decision"`
	Decision         string   `json:"decision"`
	Reason           string   `json:"reason"`
	ExpectedMatch    bool     `json:"expected_match"`
	Unknown          *Unknown `json:"unknown,omitempty"`
}

type StageMetric struct {
	Stage      string `json:"stage"`
	WallMS     int64  `json:"wall_ms"`
	PeakRSSKiB int64  `json:"peak_rss_kib"`
}

type TestMetrics struct {
	Total    int `json:"total"`
	Selected int `json:"selected"`
	Executed int `json:"executed"`
	Reused   int `json:"reused"`
	Failed   int `json:"failed"`
	Unknown  int `json:"unknown"`
}

type Inventory struct {
	DescendantDirs    int `json:"descendant_dirs"`
	RegularFiles      int `json:"regular_files"`
	GoFiles           int `json:"go_files"`
	GoPhysicalLines   int `json:"go_physical_lines"`
	GoooFiles         int `json:"gooo_files"`
	GoooPhysicalLines int `json:"gooo_physical_lines"`
	RootREADMEExcluded bool `json:"root_readme_excluded"`
}

type GeneratedInventory struct {
	Artifacts int `json:"generated_artifacts"`
	Bytes     int64 `json:"generated_bytes"`
}

type AuthorityCounts struct {
	RepositoryWrites          int `json:"repository_writes"`
	InputSourceWrites         int `json:"input_source_writes"`
	RuntimeCommitMergeTagRelease int `json:"runtime_commit_merge_tag_release"`
	CallerOwnedOutputWrites   int `json:"caller_owned_output_writes"`
	GitHubActionsArtifactUploads int `json:"github_actions_artifact_uploads"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}

type OperatorActions struct {
	RepositoryCreate int `json:"repository_create"`
	PRCreate        int `json:"pr_create"`
	Merge           int `json:"merge"`
	AnnotatedTag    int `json:"annotated_tag"`
	ReleaseCreate   int `json:"release_create"`
	AssetUploads    int `json:"asset_uploads"`
}

type AuthorityReport struct {
	Local    AuthorityCounts `json:"local"`
	Remote   AuthorityCounts `json:"remote"`
	Operator OperatorActions `json:"operator"`
}

type MetricsReport struct {
	Stages    []StageMetric       `json:"stages"`
	Tests     TestMetrics         `json:"tests"`
	Inventory Inventory           `json:"inventory"`
	Generated GeneratedInventory  `json:"generated"`
}

type ConformanceReport struct {
	Schema             string          `json:"schema"`
	Decision           string          `json:"decision"`
	Reason             string          `json:"reason"`
	Precedence         []string        `json:"precedence"`
	DenominatorID      string          `json:"denominator_id"`
	FixedDenominator   int             `json:"fixed_denominator"`
	Cases              []CaseResult    `json:"cases"`
	DecisionCounts     map[string]int  `json:"decision_counts"`
	FixedPoint         FixedPointProof `json:"fixed_point"`
	Metrics            MetricsReport   `json:"metrics"`
	Authority          AuthorityReport `json:"authority"`
	Utility            Claim           `json:"utility"`
	Improvement        Claim           `json:"improvement"`
	LocalValidationCommands int        `json:"local_validation_commands"`
	OperationalRefuted bool           `json:"operational_refuted"`
	EndToEnd           []string        `json:"end_to_end"`
}

func isDecision(value string) bool {
	return value == DecisionClosed || value == DecisionUnknown || value == DecisionRefuted
}

func decisionRank(value string) int {
	for index, candidate := range DecisionPrecedence {
		if candidate == value {
			return index
		}
	}
	return len(DecisionPrecedence)
}

func resolveDecision(values ...string) string {
	best := DecisionClosed
	for _, value := range values {
		if decisionRank(value) < decisionRank(best) {
			best = value
		}
	}
	return best
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}
