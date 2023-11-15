package emulator

// Config contains configuration for the emulator Publisher and Subscriber.
type Config struct {
	ClientURL   string `yaml:"client_url"`
	Port        string `yaml:"port"`
	PublicURL   string `yaml:"public_url"`
	TickSeconds int64  `yaml:"tick_seconds"`
	Random      bool   `yaml:"random"`
}

// Measurement is data structure containing measurements received in a single transaction.
type Measurement struct {
	Volts int64 `json:"volts"`
	Mamps int64 `json:"m_amps"`
	Power int64 `json:"power"`
}
