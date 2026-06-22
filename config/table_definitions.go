package config

type TableDefinitions struct {
	FileTransfer FileTransferTableDefinition `yaml:"file_transfer"`
	FileCheck    FileCheckTableDefinition    `yaml:"file_check"`
}

type FileTransferTableDefinition struct {
	Sheet  string `yaml:"sheet"`
	SrcCol string `yaml:"src_column"`
	DstCol string `yaml:"dst_column"`
}

type FileCheckTableDefinition struct {
	Sheet      string `yaml:"sheet"`
	NewFileCol string `yaml:"new_file_column"`
	OldFileCol string `yaml:"old_file_column"`
}
