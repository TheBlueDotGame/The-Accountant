package natsclient

// Publisher provides functionality to push messages to the pub/sub queue
type Publisher struct {
	socket
}

// PublisherConnect connects publisher to the pub/sub queue using provided config
func PublisherConnect(cfg Config) (Publisher, error) {
	var p Publisher
	var err error
	p.socket, err = connect(cfg)
	return p, err
}
