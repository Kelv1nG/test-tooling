package templates

type PageData struct {
	ListenAddr          string
	DefinitionsPath     string
	WorkbookPath        string
	ReportsRoot         string
	LoadedAt            string
	ActiveTab           string
	HasConfig           bool
	LoadError           string
	SaveMessage         string
	SaveHasErrors       bool
	TransferCount       int
	TransferPage        int
	CheckCount          int
	CheckPage           int
	TransferRows        []TransferRowView
	CheckRows           []CheckRowView
	Strategy            string
	ReferenceDate       string
	CheckReferenceDate  string
	TransferMessage     string
	TransferHasErrors   bool
	TransferSummary     TransferSummaryView
	TransferResults     []TransferResultView
	TransferSummaryRows []TransferSummaryRowView
	CheckMessage        string
	CheckHasIssues      bool
	CheckSummary        CheckSummaryView
	CheckSummaryRows    []CheckSummaryRowView
	CheckRunID          string
	CheckRunRunning     bool
	CheckRunCompleted   int
	CheckRunTotal       int
}

type TransferRowView struct {
	Index        int
	ExcelRow     int
	Src          string
	ResolvedSrc  string
	SrcExists    bool
	Dest         string
	ResolvedDest string
	DestExists   bool
	Status       string
	Badge        string
	Detail       string
}

type CheckRowView struct {
	Index               int
	ExcelRow            int
	ID                  string
	File                string
	ResolvedFile        string
	FileExists          bool
	CompareOffsetMonths int
	ResolvedCompareFile string
	CompareExists       bool
	Rules               []CheckRuleView
	Status              string
	Badge               string
	Detail              string
}

type CheckRuleView struct {
	Index           int
	ExcelRow        int
	ID              string
	CheckID         string
	Name            string
	Type            string
	Enabled         bool
	Sheet           string
	Anchor          string
	ParentDirection string
	MaxHeaderDepth  string
	RequireOrder    bool
	ScanSelect      string
	ExpectedText    string
	CompareAs       string
	Status          string
	Badge           string
	Detail          string
}

type TransferResultView struct {
	Index        int
	Src          string
	ResolvedSrc  string
	Dest         string
	ResolvedDest string
	Status       string
	Badge        string
	Detail       string
}

type TransferSummaryRowView struct {
	Index       int
	Status      string
	Badge       string
	Source      string
	Destination string
	Detail      string
}

type TransferSummaryView struct {
	Attempted   int
	Created     int
	Overwritten int
	Skipped     int
	Errors      int
	HasRun      bool
}

type CheckSummaryView struct {
	Attempted int
	Matched   int
	Changed   int
	Errors    int
	Skipped   int
	HasRun    bool
}

type CheckSummaryRowView struct {
	CheckIndex  int
	CheckID     string
	RuleID      string
	RuleName    string
	RuleType    string
	Status      string
	Badge       string
	CurrentFile string
	CompareFile string
	Sheet       string
	Detail      string
}
