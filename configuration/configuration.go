package configuration

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/bartossh/Computantis/bookkeeping"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/emulator"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/repository"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/validator"
	"github.com/bartossh/Computantis/walletapi"
	"github.com/bartossh/Computantis/zincaddapter"
)

// Configuration is the main configuration of the application that corresponds to the *.yaml file
// that holds the configuration.
type Configuration struct {
	Server        server.Config         `yaml:"server"`
	StorageConfig StorageConfig         `yaml:"storage_config"`
	Client        walletapi.Config      `yaml:"client"`
	FileOperator  fileoperations.Config `yaml:"file_operator"`
	ZincLogger    zincaddapter.Config   `yaml:"zinc_logger"`
	Validator     validator.Config      `yaml:"validator"`
	Emulator      emulator.Config       `yaml:"emulator"`
	DataProvider  dataprovider.Config   `yaml:"data_provider"`
	Bookkeeper    bookkeeping.Config    `yaml:"bookkeeper"`
}

type StorageConfig struct {
	TransactionDatabase     repository.DBConfig `yaml:"transaction_database"`
	BlockchainDatabase      repository.DBConfig `yaml:"blockchain_database"`
	NodeRegisterDatabase    repository.DBConfig `yaml:"node_register_database"`
	AddressDatabase         repository.DBConfig `yaml:"address_database"`
	TokenDatabase           repository.DBConfig `yaml:"token_database"`
	ValidatorStatusDatabase repository.DBConfig `yaml:"validator_status_database"`
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
