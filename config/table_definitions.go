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
	Sheet              string `yaml:"sheet"`
	NewFileCol         string `yaml:"new_file_column"`
	OldFileCol         string `yaml:"old_file_column"`
	HeaderSheetCol     string `yaml:"header_sheet_column"`
	AnchorCol          string `yaml:"anchor_column"`
	ParentDirectionCol string `yaml:"parent_direction_column"`
	MaxHeaderDepthCol  string `yaml:"max_header_depth_column"`
	RequireOrderCol    string `yaml:"require_order_column"`
}
