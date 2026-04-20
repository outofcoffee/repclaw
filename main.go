package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/outofcoffee/repclaw/internal/client"
	"github.com/outofcoffee/repclaw/internal/config"
	"github.com/outofcoffee/repclaw/internal/tui"
	"github.com/outofcoffee/repclaw/internal/version"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("repclaw %s\n", version.Version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	c, err := client.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "client error: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := c.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "connection error: %v\n", err)
		os.Exit(1)
	}

	app := tui.NewApp(c)
	p := tea.NewProgram(app)

	// Pump gateway events into the bubbletea program from a dedicated goroutine.
	// This is more reliable than a cmd-chain that must be re-issued after each event.
	go func() {
		for ev := range c.Events() {
			p.Send(tui.GatewayEventMsg(ev))
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
