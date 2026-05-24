package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Vixel2006/panoptes/internal/adapters"
	"github.com/Vixel2006/panoptes/internal/infra/tls"
	"github.com/Vixel2006/panoptes/internal/infra/transport"
)

func main() {
	caCert, err := tls.LoadOrGenerateCA("certs")
	if err != nil {
		log.Fatal(err)
	}

	certGen, err := tls.NewCertificateGenerator(caCert)
	if err != nil {
		log.Fatal(err)
	}

	certPath, _ := filepath.Abs("certs/panoptes-ca.crt")
	if _, err := os.Stat(certPath); err == nil {
		fmt.Printf("Root CA certificate: %s\n", certPath)
		fmt.Println("Install this CA in your browser/OS to intercept HTTPS traffic.")
		fmt.Println("On Linux:   sudo cp certs/panoptes-ca.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates")
		fmt.Println("On macOS:   sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain certs/panoptes-ca.crt")
		fmt.Println("In Firefox: Preferences → Privacy & Security → Certificates → View Certificates → Authorities → Import")
		fmt.Println()
	}

	adapter := adapter.NewInterceptAdapter(certGen)
	server := transport.NewServer("localhost", 8080, adapter.HandleConn)
	ctx := context.Background()

	fmt.Printf("Proxy listening on localhost:8080\n")

	server.Start(ctx)
}
