// Package app runs the lucinate Bubble Tea program with pluggable I/O.
//
// The CLI entry point in main.go is a thin wrapper around Run; embedders
// that need to host the program with their own input source or output sink
// (for example, tests or alternative front-ends) construct a *client.Client,
// connect it, and then call Run with a RunOptions value.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/lucinate-ai/lucinate/internal/client"
	"github.com/lucinate-ai/lucinate/internal/tui"
)

// RunOptions configures a single Run invocation.
type RunOptions struct {
	// Client is the already-connected gateway client whose events drive the
	// UI. Run does not call Connect or Close on the client; lifecycle is the
	// caller's responsibility.
	Client *client.Client

	// Input is the source of user input bytes. If nil, os.Stdin is used.
	Input io.Reader

	// Output is the destination for rendered frames. If nil, os.Stdout is
	// used.
	Output io.Writer
}

// Run starts the Bubble Tea program and blocks until it exits or ctx is
// cancelled. The events-pump goroutine that bridges gateway events into the
// program is owned by Run and stops when the program exits or ctx is
// cancelled, whichever comes first.
//
// Run never closes the client; the caller is expected to Close it after Run
// returns.
func Run(ctx context.Context, opts RunOptions) error {
	if opts.Client == nil {
		return errors.New("app.Run: Client is required")
	}
	in := opts.Input
	if in == nil {
		in = os.Stdin
	}
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	model := tui.NewApp(opts.Client)
	p := tea.NewProgram(model,
		tea.WithContext(runCtx),
		tea.WithInput(in),
		tea.WithOutput(out),
	)

	pumpDone := make(chan struct{})
	go func() {
		defer close(pumpDone)
		events := opts.Client.Events()
		for {
			select {
			case ev, ok := <-events:
				if !ok {
					return
				}
				p.Send(tui.GatewayEventMsg(ev))
			case <-runCtx.Done():
				return
			}
		}
	}()

	_, err := p.Run()
	cancel()
	<-pumpDone

	if err != nil {
		return fmt.Errorf("program: %w", err)
	}
	return nil
}
