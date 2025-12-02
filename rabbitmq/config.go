package rabbitmq

import "time"

// Coonnection holds the settings for the RabbitMQ connection
type Config struct {
	// URL is the AMQP connection string (amqp://user:pass@host:port/vhost)
	URL string

	// AppName is used for metadata in the connection (optional)
	AppName string

	// ReconnectDelay is the time to wait between reconnection attempts.
	ReconnectDelay time.Duration

	// MaxRetries is the number of connection attempts before emitting a fatal error.
	MaxRetries int
}

// Consumer holds settings specific to consuming messages
type Consumer struct {
	// ConsumerName identifies this specific consumer instance in RabbitMQ.
	Name string

	// Queue is the name of the pre-existing queue to consume from.
	Queue string

	// PrefetchCount limits how many unacknowledged messages the server delivers.
	// Critical for load balancing and preventing memory overflows.
	PrefetchCount int

	// Workers is the number of concurrent goroutines processing messages.
	Workers int

	// RetryMax is the number of times to retry the handler before Nacking/DLQing.
	RetryMax int

	// RetryStart is the initial duration for exponential backoff (e.g., 100ms).
	RetryStart time.Duration
}
