package config

type Configuration struct {
	FileTransferMaps []FileTransferMap
	FileCheckConfigs []FileCheckConfig
}

type FileTransferMap struct {
	ExcelRow int
	Src      string
	Dest     string
}

type FileCheckConfig struct {
	ExcelRow            int
	ID                  string
	File                string
	CompareOffsetMonths int
	Rules               []VerificationRule
}

type VerificationRuleType string

const (
	VerificationRuleTypeHeaderCompare VerificationRuleType = "header_compare"
	VerificationRuleTypeExactText     VerificationRuleType = "exact_text"
	VerificationRuleTypeAnchorScan    VerificationRuleType = "anchor_scan_match"
)

type VerificationRule struct {
	ExcelRow      int
	ID            string
	CheckID       string
	Name          string
	Type          VerificationRuleType
	Enabled       bool
	HeaderCompare HeaderCheckConfig
	ExactText     ExactMatchCheckConfig
	AnchorScan    AnchorScanMatchConfig
}

type HeaderCheckConfig struct {
	Sheet           string `json:"sheet"`
	Anchor          string `json:"anchor"`
	ParentDirection string `json:"parent_direction"`
	MaxHeaderDepth  int    `json:"max_header_depth"`
	RequireOrder    bool   `json:"require_order"`
}

type ExactMatchCheckConfig struct {
	Sheet        string `json:"sheet"`
	ExpectedText string `json:"expected_text"`
}

type AnchorScanMatchConfig struct {
	Sheet        string `json:"sheet"`
	Anchor       string `json:"anchor"`
	Direction    string `json:"direction"`
	Select       string `json:"select"`
	ExpectedText string `json:"expected_text"`
	CompareAs    string `json:"compare_as"`
}

const (
	AnchorScanSelectLastNonEmptyBeforeBlank = "last_non_empty_before_blank"

	AnchorScanCompareExactText = "exact_text"
	AnchorScanCompareDate      = "date"
)
