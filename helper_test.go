package vio_test

import (
	"context"
	"time"
)

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func deadlineExceededContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), -time.Second)
	cancel()
	return ctx
}
