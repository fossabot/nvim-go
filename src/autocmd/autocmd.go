// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autocmd

import (
	"context"
	"sync"

	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
	"github.com/zchee/nvim-go/src/buildctx"
	"github.com/zchee/nvim-go/src/command"
	"github.com/zchee/nvim-go/src/logger"
	"golang.org/x/sync/syncmap"
)

// Autocmd represents a autocmd context.
type Autocmd struct {
	ctx    context.Context
	cancel context.CancelFunc

	Nvim         *nvim.Nvim
	buildContext *buildctx.Context
	cmd          *command.Command

	bufWritePostChan chan error
	bufWritePreChan  chan interface{}
	mu               sync.Mutex
	wg               sync.WaitGroup

	errs *syncmap.Map
}

// Register registers autocmd to Neovim.
func Register(pctx context.Context, p *plugin.Plugin, buildContext *buildctx.Context, cmd *command.Command) {
	ctx, cancel := context.WithCancel(pctx)
	ctx = logger.NewContext(ctx, logger.FromContext(ctx).Named("autocmd"))

	autocmd := &Autocmd{
		ctx:              ctx,
		cancel:           cancel,
		Nvim:             p.Nvim,
		buildContext:     buildContext,
		cmd:              cmd,
		bufWritePreChan:  make(chan interface{}),
		bufWritePostChan: make(chan error),
		errs:             new(syncmap.Map),
	}

	logger.FromContext(ctx).Info("Register")

	// Handle the initial start Neovim process.
	// Note that does not run the 'VimEnter' handler if open the *not* go file. Because 'VimEnter' handler already run the other file or directory.
	// TODO(zchee): consider Pattern to '*' instead of '*.go' with get '&filetype' and early return
	// p.HandleAutocmd(&plugin.AutocmdOptions{Event: "VimEnter", Pattern: "*", Group: "nvim-go", Eval: "*"}, autocmd.VimEnter)

	// Handle the open the file.
	// If open the file at first, run the 'BufEnter' -> 'VimEnter'.
	p.HandleAutocmd(&plugin.AutocmdOptions{Event: "BufEnter", Pattern: "*.go", Group: "nvim-go", Eval: "*"}, autocmd.BufEnter)

	// BufNewFile: Handle create the new file.
	// BufReadPre: Handle the before the read to file.
	// If create the new file, does not run the 'BufReadPre', Instead of 'BufNewFile'.
	p.HandleAutocmd(&plugin.AutocmdOptions{Event: "BufNewFile,BufReadPre", Group: "nvim-go-autocmd", Pattern: "*.go", Eval: "*"}, autocmd.BufReadPre)

	// p.HandleAutocmd(&plugin.AutocmdOptions{Event: "WinEnter", Group: "nvim-go-autocmd", Pattern: "*.go", Eval: "*"}, autocmd.WinEnter)

	// Handle the before the write to file.
	p.HandleAutocmd(&plugin.AutocmdOptions{Event: "BufWritePre", Pattern: "*.go", Group: "nvim-go", Eval: "*"}, autocmd.bufWritePre)

	// Handle the after the write to file.
	p.HandleAutocmd(&plugin.AutocmdOptions{Event: "BufWritePost", Pattern: "*.go", Group: "nvim-go", Eval: "*"}, autocmd.bufWritePost)
}
