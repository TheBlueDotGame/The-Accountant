package emulator

// Config contains configuration for the emulator Publisher and Subscriber.
type Config struct {
	ClientURL      string `yaml:"client_url"`
	Port           string `yaml:"port"`
	PublicURL      string `yaml:"public_url"`
	TimeoutSeconds int64  `yaml:"timeout_seconds"`
	TickSeconds    int64  `yaml:"tick_seconds"`
	Random         bool   `yaml:"random"`
}

// Measurement is data structure containing measurements received in a single transaction.
type Measurement struct {
	Volts int `json:"volts"`
	Mamps int `json:"m_amps"`
	Power int `json:"power"`
}
