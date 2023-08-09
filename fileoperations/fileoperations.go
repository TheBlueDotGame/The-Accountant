package fileoperations

// Config holds configuration of the file operator Helper.
type Config struct {
	WalletPath   string `yaml:"wallet_path"`   // wallet path to the wallet file
	WalletPasswd string `yaml:"wallet_passwd"` // wallet password to the wallet file in hex format
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
