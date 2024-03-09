package fileoperations

// Config holds configuration of the file operator Helper.
type Config struct {
	WalletPath    string `yaml:"wallet_path"`   // wallet path to the wallet gob file
	WalletPasswd  string `yaml:"wallet_passwd"` // password to the wallet gob file in hex format
	WalletPemPath string `yaml:"pem_path"`      // path to ed25519 pem file
	CAPath        string `yaml:"ca_cert"`       // path to ed25519 pem file
}

// Helper holds all file operation methods.
type Helper struct {
	s   Sealer
	cfg Config
}

// New creates new Helper.
func New(cfg Config, s Sealer) Helper {
	return Helper{
		cfg: cfg,
		s:   s,
	}
}
