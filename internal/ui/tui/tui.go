package tui

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dc/cr2/internal/export"
	"github.com/dc/cr2/internal/reviewapp"
)

type Options struct {
	OutputFile string
}

type UI struct {
	app  reviewapp.App
	spec reviewapp.DiffSpec
	opts Options
}

func New(app reviewapp.App, spec reviewapp.DiffSpec, opts Options) UI {
	return UI{app: app, spec: spec, opts: opts}
}

func (ui UI) Run(ctx context.Context) error {
	session, err := ui.app.Load(ctx, ui.spec)
	if err != nil {
		return err
	}
	m := newModel(ctx, ui.app, session, ui.opts)
	final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	fm, ok := final.(model)
	if !ok || !fm.saved {
		return nil
	}
	md := export.Markdown(fm.session)
	if fm.opts.OutputFile == "" {
		fmt.Print(md)
		return nil
	}
	if err := os.WriteFile(fm.opts.OutputFile, []byte(md), 0644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Review saved to %s\n", fm.opts.OutputFile)
	return nil
}
