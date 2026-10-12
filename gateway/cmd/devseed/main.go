// Command devseed writes the design's number labels into an empty gateway
// database so the UI can be previewed against `fakesignal -seed design`.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
)

func main() {
	path := flag.String("db", "", "path to the gateway SQLite database")
	flag.Parse()
	if *path == "" {
		log.Fatal("-db is required")
	}

	ctx := context.Background()
	x, err := db.Open(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer x.Close()

	rows, err := db.ListNumbers(ctx, x)
	if err != nil {
		log.Fatal(err)
	}
	if len(rows) > 0 {
		log.Printf("numbers table already has %d rows; nothing to do", len(rows))
		return
	}

	for _, n := range [][2]string{
		{"+27825550143", "Reception"},
		{"+27715550198", "Pastoral care line"},
		{"+27605550127", "Events"},
		{"+27835550110", "Old helpdesk"},
	} {
		if err := db.InsertNumber(ctx, x, n[0], n[1]); err != nil {
			log.Fatal(err)
		}
	}
	// Events is awaiting its code; Old helpdesk stays a bare row (not registered).
	now := time.Now().UTC()
	if err := db.SetRegisterRequested(ctx, x, "+27605550127", &now); err != nil {
		log.Fatal(err)
	}
	log.Print("seeded 4 numbers")
}
