package configuration

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/bartossh/Computantis/accountant"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/emulator"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/gossip"
	"github.com/bartossh/Computantis/natsclient"
	"github.com/bartossh/Computantis/notaryserver"
	"github.com/bartossh/Computantis/walletapi"
	"github.com/bartossh/Computantis/webhooksserver"
	"github.com/bartossh/Computantis/zincaddapter"
)

// Configuration is the main configuration of the application that corresponds to the *.yaml file
// that holds the configuration.
type Configuration struct {
	NotaryServer   notaryserver.Config   `yaml:"notary_server"`
	Gossip         gossip.Config         `yaml:"gossip_server"`
	Accountant     accountant.Config     `yaml:"accountant"`
	Nats           natsclient.Config     `yaml:"nats"`
	FileOperator   fileoperations.Config `yaml:"file_operator"`
	ZincLogger     zincaddapter.Config   `yaml:"zinc_logger"`
	DataProvider   dataprovider.Config   `yaml:"data_provider"`
	WebhooksServer webhooksserver.Config `yaml:"webhooks_server"`
	Emulator       emulator.Config       `yaml:"emulator"`
	Client         walletapi.Config      `yaml:"client"`
	IsProfiling    bool                  `yaml:"is_profiling"` // Indicates if node server is running in profiling mode and will create `default.pgo` file.
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
