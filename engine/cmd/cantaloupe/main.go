package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/xmasdev/Cantaloupe/engine"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <torrent-file> <output-directory>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	torrentPath := os.Args[1]
	outputDir := os.Args[2]

	client, err := engine.NewEngine(torrentPath, outputDir)
	if err != nil {
		fatal(err)
	}
	defer client.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go client.AcceptPeers(ctx)

	fmt.Printf("downloading %s to %s (peer port %d)\n", client.Metadata.Info.Name, outputDir, client.Port)
	if err := client.Start(); err != nil {
		fatal(err)
	}

	fmt.Println("download complete")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "cantaloupe:", err)
	os.Exit(1)
}
