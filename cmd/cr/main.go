package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/dc/cr2/internal/gitrepo"
	"github.com/dc/cr2/internal/reviewapp"
	"github.com/dc/cr2/internal/ui/tui"
)

func main() {
	var (
		output   string
		branch   bool
		unstaged bool
	)
	flag.StringVar(&output, "output", "", "write review markdown to file on save")
	flag.StringVar(&output, "o", "", "write review markdown to file on save")
	flag.BoolVar(&branch, "branch", false, "diff current branch against main/master")
	flag.BoolVar(&branch, "b", false, "diff current branch against main/master")
	flag.BoolVar(&unstaged, "unstaged", false, "review only unstaged tracked changes plus untracked files")
	flag.BoolVar(&unstaged, "u", false, "review only unstaged tracked changes plus untracked files")
	flag.Usage = usage
	flag.Parse()

	if unstaged && branch {
		exitUsage("--unstaged cannot be combined with --branch")
	}
	if flag.NArg() > 1 {
		exitUsage("expected at most one revision argument")
	}

	ctx := context.Background()
	repo := gitrepo.NewLocal()
	spec := reviewapp.DiffSpec{Mode: reviewapp.DiffWorkingTree}

	if unstaged {
		spec.Mode = reviewapp.DiffUnstaged
	} else if flag.NArg() == 1 {
		spec.Mode = reviewapp.DiffRevision
		spec.RevSpec = flag.Arg(0)
	} else if branch {
		if hasCommits, err := repo.HasCommits(ctx); err == nil && hasCommits {
			spec.Mode = reviewapp.DiffRevision
			spec.RevSpec = repo.DefaultBranch(ctx) + "...HEAD"
		}
	} else if cur, err := repo.CurrentBranch(ctx); err == nil {
		def := repo.DefaultBranch(ctx)
		hasCommits, _ := repo.HasCommits(ctx)
		if hasCommits && cur != "" && cur != "HEAD" && cur != def {
			spec.Mode = reviewapp.DiffRevision
			spec.RevSpec = def + "...HEAD"
		}
	}

	app := reviewapp.New(repo)
	ui := tui.New(app, spec, tui.Options{OutputFile: output})
	if err := ui.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "cr2: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `cr2 - clean-room code review CLI

Usage:
  cr2                    Review current branch or working tree
  cr2 --branch           Review branch changes against main/master
  cr2 --unstaged         Review unstaged changes
  cr2 REV                Review a commit or revision range
  cr2 -o REVIEW.md       Save markdown to REVIEW.md when using save

`)
	flag.PrintDefaults()
}

func exitUsage(msg string) {
	fmt.Fprintf(os.Stderr, "cr2: %s\n\n", msg)
	usage()
	os.Exit(2)
}
