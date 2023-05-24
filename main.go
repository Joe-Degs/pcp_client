package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/Joe-Degs/pcp_client/client"
)

var (
	pcpAddr     = flag.String("pcp.addr", "pool:9898", "address of the pcp backend")
	clientAddr  = flag.String("client.addr", "localhost:8080", "address for this pcp client")
	pcpUsername = flag.String("pcp.username", "pgpool", "username of user to use for authorization")
	pcpPassword = flag.String("pcp.password", "password", "password to use for authorization")
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	flag.Parse()

	cli, err := client.NewClient(ctx, *pcpAddr, *pcpUsername, *pcpPassword)
	if err != nil {
		log.Fatalf("could not open connection to pcp, %s", err)
	}

	if err := cli.Authorize(); err != nil {
		log.Fatal(err)
	}

	if _, err := cli.NodeCount(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello bitches!\n")
	})

	go func() {
		<-ctx.Done()
		cli.Close()
	}()

	if err := http.ListenAndServe(*clientAddr, nil); err != nil {
		log.Fatalf("cannot start pcp_client: %s", err)
	}
}
