package templates

type PageData struct {
	ListenAddr        string
	DefinitionsPath   string
	WorkbookPath      string
	LoadedAt          string
	ActiveTab         string
	HasConfig         bool
	LoadError         string
	SaveMessage       string
	SaveHasErrors     bool
	TransferCount     int
	CheckCount        int
	TransferRows      []TransferRowView
	CheckRows         []CheckRowView
	Strategy          string
	ReferenceDate     string
	TransferMessage   string
	TransferHasErrors bool
	TransferSummary   TransferSummaryView
	TransferResults   []TransferResultView
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
	Index     int
	ExcelRow  int
	NewFile   string
	NewExists bool
	OldFile   string
	OldExists bool
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
