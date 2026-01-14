package tools

/*
This is the global configuration or global variable
*/

type ServiceConfig struct {
	Port     int16 `yaml:"port"`
	Database struct {
		DataBaseType string `yaml:"databaseType"`
		ConnPath     string `yaml:"connPath"`
		//Type         string     `yaml:"type"`
		Path        string     `yaml:"path"`
		Host        string     `yaml:"host"`
		Port        int16      `yaml:"port"`
		AuthSource  string     `yaml:"authSource"`
		AuthType    string     `yaml:"authType"`
		Description UserConfig `yaml:"description"`
		BaseName    string     `yaml:"basename"`
	}
	Login struct {
		User UserConfig
	} `yaml:"login"`
}

type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
