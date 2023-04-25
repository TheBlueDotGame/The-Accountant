package configuration

import (
	"fmt"
	"os"

	"github.com/bartossh/Computantis/bookkeeping"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/repo"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/validator"
	"gopkg.in/yaml.v2"
)

// Configuration is the main configuration of the application that corresponds to the *.yaml file
// that holds the configuration.
type Configuration struct {
	Bookkeeper   bookkeeping.Config  `yaml:"bookkeeper"`
	Server       server.Config       `yaml:"server"`
	Database     repo.Config         `yaml:"database"`
	DataProvider dataprovider.Config `yaml:"data_provider"`
	Validator    validator.Config    `yaml:"validator"`
}

// Read reads the configuration  from the file and returns the MainYaml struct.
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
