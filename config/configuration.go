package config

type Configuration struct {
	FileTransferMaps []FileTransferMap
	FileCheckRules   []FileCheckRule
}

type FileTransferMap struct {
	Src  string
	Dest string
}

type FileCheckRule struct {
	NewFile string
	OldFile string
}
