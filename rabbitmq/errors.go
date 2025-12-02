package rabbitmq

import "errors"

var (
	ErrNotConnected = errors.New("rabbitmq: client not connected")
	ErrShutdown     = errors.New("rabbitmq: client is shutting down")
)
