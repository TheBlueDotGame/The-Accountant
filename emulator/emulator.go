package emulator

// Config contains configuration for the emulator Publisher and Subscriber.
type Config struct {
	TimeoutSeconds int64  `yaml:"timeout_seconds"`
	TickSeconds    int64  `yaml:"tick_seconds"`
	Random         bool   `yaml:"random"`
	ClientURL      string `yaml:"client_url"`
	Port           string `yaml:"port"`
	PublicURL      string `yaml:"public_url"`
}
