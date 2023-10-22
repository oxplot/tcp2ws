package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/oxplot/tcp2ws"
)

var (
	listen = flag.String("listen", ":8080", "websocket listen address:port")
	usage  = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] addr:port\n", os.Args[0])
		flag.PrintDefaults()
	}
	tcpAddr string
)

func run() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tcp2ws.ForwardWebsocket(w, r, tcpAddr)
	})
	log.Printf("listening on ws://%s", *listen)
	return http.ListenAndServe(*listen, nil)

}

func main() {
	log.SetFlags(0)
	flag.Parse()
	if flag.Arg(0) == "" {
		usage()
		os.Exit(1)
	}
	tcpAddr = flag.Arg(0)
	if err := run(); err != nil {
		log.Fatalf("%s\n", err)
	}
}
