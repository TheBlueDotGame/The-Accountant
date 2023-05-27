package emulator

// Config contains configuration for the emulator Publisher and Subscriber.
type Config struct {
	TimeoutSeconds              int64  `yaml:"timeout_seconds"`
	TickSeconds                 int64  `yaml:"tick_seconds"`
	Random                      bool   `yaml:"random"`
	SignerServiceAddress        string `yaml:"signer_service_address"`
	ValidatorCreateHookEndpoint string `yaml:"validator_create_hook_endpoint"`
	Port                        string `yaml:"port"`
}
