package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"

	"github.com/oxplot/tcp2ws"
)

var (
	listen = flag.String("listen", ":7101", "TCP listen address:port")
	usage  = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] ws[s]://...\n", os.Args[0])
		flag.PrintDefaults()
	}
	wsURL *url.URL
)

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Stop on Ctrl-C

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		log.Printf("interrupted")
		cancel()
	}()

	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}
	go func() {
		<-ctx.Done()
		lis.Close()
	}()
	log.Printf("listening on %s", lis.Addr())

	h := tcp2ws.NewTCPHandler(wsURL.String())
	for {
		conn, err := lis.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil // when lis is closed
			}
			return fmt.Errorf("accept error: %w", err)
		}
		log.Printf("handling connection from %s", conn.RemoteAddr())
		go func() {
			if err := h.Handle(ctx, conn); err != nil {
				log.Printf("handle error: %v", err)
			}
		}()
	}
}

func main() {
	log.SetFlags(0)

	var err error
	flag.Parse()
	if flag.Arg(0) == "" {
		usage()
		os.Exit(1)
	}
	wsURL, err = url.Parse(flag.Arg(0))
	if err != nil {
		log.Fatalf("url parse error: %s", err)
	}
	if wsURL.Scheme != "ws" && wsURL.Scheme != "wss" {
		log.Fatalf("url scheme must be ws or wss")
	}
	if wsURL.Host == "" {
		log.Fatalf("url host must be set")
	}
	if err := run(); err != nil {
		log.Fatalf("%s\n", err)
	}
}
