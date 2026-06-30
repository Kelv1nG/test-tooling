package templates

type PageData struct {
	ListenAddr         string
	DefinitionsPath    string
	WorkbookPath       string
	LoadedAt           string
	ActiveTab          string
	HasConfig          bool
	LoadError          string
	SaveMessage        string
	SaveHasErrors      bool
	TransferCount      int
	CheckCount         int
	TransferRows       []TransferRowView
	CheckRows          []CheckRowView
	Strategy           string
	ReferenceDate      string
	CheckReferenceDate string
	TransferMessage    string
	TransferHasErrors  bool
	TransferSummary    TransferSummaryView
	TransferResults    []TransferResultView
	CheckMessage       string
	CheckHasIssues     bool
	CheckSummary       CheckSummaryView
}

type TransferRowView struct {
	Index      int
	ExcelRow   int
	Src        string
	SrcExists  bool
	Dest       string
	DestExists bool
	Status     string
	Badge      string
	Detail     string
}

type CheckRowView struct {
	Index               int
	ExcelRow            int
	ID                  string
	File                string
	FileExists          bool
	CompareOffsetMonths int
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
	ExpectedText    string
	Status          string
	Badge           string
	Detail          string
}

type TransferResultView struct {
	Index  int
	Src    string
	Dest   string
	Status string
	Badge  string
	Detail string
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
