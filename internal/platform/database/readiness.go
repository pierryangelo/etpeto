package database

import "context"

type Pinger interface {
	Ping(context.Context) error
}

type Checker struct {
	Pinger Pinger
}

func (c Checker) Ready(ctx context.Context) error {
	return c.Pinger.Ping(ctx)
}
