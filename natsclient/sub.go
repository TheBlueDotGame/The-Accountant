package natsclient

// Subscriber provides functionality to pull messages from the pub/sub queue.
type Subscriber struct {
	socket
}

// SubscriberConnect connects publisher to the pub/sub queue using provided config
func SubscriberConnect(cfg Config) (Subscriber, error) {
	var s Subscriber
	var err error
	s.socket, err = connect(cfg)
	return s, err
}
