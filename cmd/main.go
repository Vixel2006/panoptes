package main

import (
	"context"

	"github.com/Vixel2006/panoptes/internal/adapters"
	"github.com/Vixel2006/panoptes/internal/infra/transport"
)

func main() {
	adapter := adapter.NewInterceptAdapter()
	server := transport.NewServer("localhost", 8080, adapter.HandleConn)
	ctx := context.Background()

	server.Start(ctx)
}
