package config

type Configuration struct {
	FileTransferMaps []FileTransferMap
	FileCheckRules   []FileCheckRule
}

type FileTransferMap struct {
	ExcelRow int
	Src      string
	Dest     string
}

type FileCheckRule struct {
	ExcelRow    int
	NewFile     string
	OldFile     string
	HeaderCheck HeaderCheckConfig
}

type HeaderCheckConfig struct {
	Sheet           string
	Anchor          string
	ParentDirection string
	MaxHeaderDepth  int
	RequireOrder    bool
}
