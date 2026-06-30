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
	Sheet      string                        `yaml:"sheet"`
	IDCol      string                        `yaml:"id_column"`
	NewFileCol string                        `yaml:"new_file_column"`
	OldFileCol string                        `yaml:"old_file_column"`
	Rules      FileCheckRulesTableDefinition `yaml:"rules"`
}

type FileCheckRulesTableDefinition struct {
	Sheet       string `yaml:"sheet"`
	CheckIDCol  string `yaml:"check_id_column"`
	RuleIDCol   string `yaml:"rule_id_column"`
	RuleNameCol string `yaml:"rule_name_column"`
	RuleTypeCol string `yaml:"rule_type_column"`
	EnabledCol  string `yaml:"enabled_column"`
	ConfigCol   string `yaml:"config_column"`
}
