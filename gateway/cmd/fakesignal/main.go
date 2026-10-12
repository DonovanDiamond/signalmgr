// Command fakesignal runs the in-memory fake signal-cli-rest-api for development.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/DonovanDiamond/signalmgr/gateway/src/fakesignal"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8081", "address to listen on")
	empty := flag.Bool("empty", false, "start without seeded numbers")
	flag.Parse()

	log.Printf("fakesignal listening on %s (seeded=%t)", *listen, !*empty)
	log.Fatal(http.ListenAndServe(*listen, fakesignal.New(!*empty)))
}
