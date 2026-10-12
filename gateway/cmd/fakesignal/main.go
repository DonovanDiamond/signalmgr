// Command fakesignal runs the in-memory fake signal-cli-rest-api for development.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/DonovanDiamond/signalmgr/gateway/src/fakesignal"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8081", "address to listen on")
	empty := flag.Bool("empty", false, "start without seeded numbers")
	seed := flag.String("seed", "", `seed set: "" for the default numbers, "design" for the UI design content`)
	format := flag.String("accounts-format", "strings", `GET /v1/accounts shape: "strings" (signal-cli-rest-api 0.101) or "objects" (PR #896, allows numberless accounts)`)
	linkAfter := flag.Duration("link-after", 8*time.Second, "delay before a new account appears after a link request (0 = never)")
	flag.Parse()

	if *seed != "" && *seed != "design" {
		log.Fatalf("unknown -seed %q (want \"design\")", *seed)
	}
	if *format != "strings" && *format != "objects" {
		log.Fatalf("unknown -accounts-format %q (want \"strings\" or \"objects\")", *format)
	}
	srv := fakesignal.New(!*empty && *seed == "")
	srv.Objects = *format == "objects"
	if *seed == "design" {
		srv.SeedDesign()
	}
	srv.LinkAfter = *linkAfter

	log.Printf("fakesignal listening on %s (empty=%t seed=%q accounts-format=%s link-after=%s)", *listen, *empty, *seed, *format, *linkAfter)
	log.Fatal(http.ListenAndServe(*listen, srv))
}
