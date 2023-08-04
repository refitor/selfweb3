package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"selfweb3/src/server"
)

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())

	server.Run(ctx, nil)

	// wait signal
	var sigch = make(chan os.Signal)
	signal.Notify(sigch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT) //, syscall.SIGUSR1, syscall.SIGUSR2)
	<-sigch
	cancel()
}
