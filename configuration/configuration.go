package configuration

import (
	"fmt"
	"os"

	"github.com/bartossh/Computantis/bookkeeping"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/validator"
	"gopkg.in/yaml.v2"
)

// Config contains configuration for the database.
type DBConfig struct {
	ConnStr      string `yaml:"conn_str"`         // ConnStr is the connection string to the database.
	DatabaseName string `yaml:"database_name"`    // DatabaseName is the name of the database.
	Token        string `yaml:"token"`            // Token is the token that is used to confirm api clients access.
	TokenExpire  int64  `yaml:"token_expiration"` // TokenExpire is the number of seconds after which token expires.
}

// Configuration is the main configuration of the application that corresponds to the *.yaml file
// that holds the configuration.
type Configuration struct {
	Bookkeeper   bookkeeping.Config    `yaml:"bookkeeper"`
	Server       server.Config         `yaml:"server"`
	Database     DBConfig              `yaml:"database"`
	DataProvider dataprovider.Config   `yaml:"data_provider"`
	Validator    validator.Config      `yaml:"validator"`
	FileOperator fileoperations.Config `yaml:"file_operator"`
}

// Read reads the configuration from the file and returns the Configuration with set fields according to the yaml setup.
func Read(path string) (Configuration, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return Configuration{}, err
	}

	var main Configuration
	err = yaml.Unmarshal(buf, &main)
	if err != nil {
		return Configuration{}, fmt.Errorf("in file %q: %w", path, err)
	}

	return main, err
}
