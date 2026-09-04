package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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
	go reportProgress(ctx, client)

	fmt.Printf("downloading %s to %s (peer port %d)\n", client.Metadata.Info.Name, outputDir, client.Port)
	if err := client.Start(); err != nil {
		fatal(err)
	}

	fmt.Println("download complete")
}

func reportProgress(ctx context.Context, client *engine.Engine) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, detail := client.Status()
			var downloaded int64
			for i := range client.Download.Pieces {
				downloaded += client.Download.Pieces[i].DownloadedBytes()
			}
			if detail != "" {
				fmt.Printf("state=%s downloaded=%s peers=%d error=%s\n", state, formatBytes(downloaded), len(client.Peers), detail)
			} else {
				fmt.Printf("state=%s downloaded=%s peers=%d\n", state, formatBytes(downloaded), len(client.Peers))
			}
		}
	}
}

func formatBytes(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	return fmt.Sprintf("%.1f MB", float64(value)/(1024*1024))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "cantaloupe:", err)
	os.Exit(1)
}
