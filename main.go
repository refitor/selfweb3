package main

import (
	"context"
	"embed"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"selfweb3/backend/src/server"
)

// go:embed rsweb
var memfs embed.FS

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())

	server.Run(ctx, &memfs)

	// wait signal
	var sigch = make(chan os.Signal)
	signal.Notify(sigch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT) //, syscall.SIGUSR1, syscall.SIGUSR2)
	<-sigch
	cancel()
}
