package main

import (
	"context"
	"fmt"

	"github.com/Vixel2006/panoptes/internal/adapters"
)

func main() {
	server := adapter.NewServer("localhost", "8080")

	server.Connect(context.Background())
}
