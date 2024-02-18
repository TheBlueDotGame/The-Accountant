package emulator

// Config contains configuration for the emulator Publisher and Subscriber.
type Config struct {
	ClientURL             string   `yaml:"client_url"`
	Port                  string   `yaml:"port"`
	PublicURL             string   `yaml:"public_url"`
	ReceiverPublicAddr    string   `yaml:"receiver_public_address"`
	NotaryNodes           []string `yaml:"notary_nodes"`
	TickMillisecond       int64    `yaml:"tick_millisecond"`
	Random                bool     `yaml:"random"`
	SpicePerTransaction   int      `yaml:"spice_per_transaction"`
	SleepInSecBeforeStart int      `yaml:"sleep_in_seconds_before_start"`
}

// Measurement is data structure containing measurements received in a single transaction.
type Measurement struct {
	Volts int64 `json:"volts"`
	Mamps int64 `json:"m_amps"`
	Power int64 `json:"power"`
}
