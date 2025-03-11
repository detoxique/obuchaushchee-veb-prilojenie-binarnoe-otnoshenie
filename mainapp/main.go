package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/detoxique/obuchaushchee-veb-prilojenie-binarnoe-otnoshenie/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		return err
	}

	return nil
}
