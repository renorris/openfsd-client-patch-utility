package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	fmt.Println("Starting openfsd client patch utility...")

	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelCtx()

	runFlow(ctx)
}
