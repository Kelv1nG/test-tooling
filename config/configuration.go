package config

type Configuration struct {
	FileTransferMaps []FileTransferMap
	FileCheckRules   []FileCheckRule
}

type FileTransferMap struct {
	ExcelRow int
	Src  string
	Dest string
}

type FileCheckRule struct {
	ExcelRow int
	NewFile string
	OldFile string
}
