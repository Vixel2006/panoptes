package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	adapter "github.com/Vixel2006/panoptes/internal/adapters"
	"github.com/Vixel2006/panoptes/internal/adapters/repo"
	"github.com/Vixel2006/panoptes/internal/app"
	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/services"
	"github.com/Vixel2006/panoptes/internal/infra/db"
	"github.com/Vixel2006/panoptes/internal/infra/tls"
	"github.com/Vixel2006/panoptes/internal/infra/transport"
	"github.com/Vixel2006/panoptes/internal/ui"
)

func main() {
	var (
		sessionFlag = flag.String("session", "", "Start or resume a session by name")
		sFlag       = flag.String("s", "", "Start or resume a session by name (shorthand)")
		listFlag    = flag.Bool("list", false, "List all available sessions and exit")
		lFlag       = flag.Bool("l", false, "List all available sessions and exit (shorthand)")
	)
	flag.Parse()

	sessionName := *sessionFlag
	if sessionName == "" {
		sessionName = *sFlag
	}
	listSess := *listFlag || *lFlag

	idGen := adapter.NewUUIDGenerator()
	decompressor := adapter.NewDecompressor()

	if listSess {
		database, err := db.Open("panoptes.db")
		if err != nil {
			log.Fatal(err)
		}
		defer database.Close()

		sessions := app.NewSessionManager(repo.NewSessionRepository(database.DB), idGen)
		sList, err := sessions.List()
		if err != nil {
			log.Fatal(err)
		}
		if len(sList) == 0 {
			fmt.Println("No sessions found.")
		} else {
			fmt.Println("Available sessions:")
			for _, s := range sList {
				fmt.Printf(" - %s (ID: %s, Created: %s)\n", s.Name, s.ID, s.CreatedAt.Format("2006-01-02 15:04:05"))
			}
		}
		return
	}

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

	database, err := db.Open("panoptes.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	requestCh := make(chan model.Request, 100)

	reqRepo := repo.NewRequestRepository(database.DB)
	respRepo := repo.NewResponseRepository(database.DB)
	interceptor := app.NewInterceptor(requestCh, reqRepo, respRepo)
	barrier := service.NewBarrier()

	a := adapter.NewInterceptAdapter(certGen, barrier, interceptor, decompressor, idGen, requestCh)

	server := transport.NewServer("localhost", 8080, a.HandleConn)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Proxy listening on localhost:8080")
		server.Start(ctx)
	}()

	sessions := app.NewSessionManager(repo.NewSessionRepository(database.DB), idGen)
	groups := app.NewGroupManager(repo.NewGroupRepository(database.DB), idGen)
	notes := app.NewNoteManager(repo.NewNoteRepository(database.DB), idGen)

	cfg := ui.Config{
		Barrier:            a.Barrier(),
		Interceptor:        a.Interceptor(),
		RequestCh:          a.RequestCh(),
		Sessions:           sessions,
		Groups:             groups,
		Notes:              notes,
		ReqRepo:            reqRepo,
		RespRepo:           respRepo,
		InitialSessionName: sessionName,
	}

	if err := ui.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
