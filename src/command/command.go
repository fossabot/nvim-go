// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package command

import (
	"context"

	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
	"github.com/zchee/nvim-go/src/buildctx"
	"github.com/zchee/nvim-go/src/command/delve"
	"github.com/zchee/nvim-go/src/logger"
	"golang.org/x/sync/syncmap"
)

// Command represents a nvim-go plugins commands.
type Command struct {
	ctx    context.Context
	cancel context.CancelFunc

	Nvim         *nvim.Nvim
	buildContext *buildctx.Context
	errs         *syncmap.Map
}

// NewCommand return the new Command type with initialize some variables.
func NewCommand(pctx context.Context, v *nvim.Nvim, buildctxt *buildctx.Context) *Command {
	ctx, cancel := context.WithCancel(pctx)
	ctx = logger.NewContext(ctx, logger.FromContext(ctx).Named("command"))

	return &Command{
		ctx:          ctx,
		cancel:       cancel,
		Nvim:         v,
		buildContext: buildctxt,
		errs:         new(syncmap.Map),
	}
}

// Register register nvim-go command or function to Neovim over the msgpack-rpc plugin interface.
func Register(ctx context.Context, p *plugin.Plugin, buildctxt *buildctx.Context) *Command {
	c := NewCommand(ctx, p.Nvim, buildctxt)

	// Register command and function
	// CommandOptions order: Name, NArgs, Range, Count, Addr, Bang, Register, Eval, Bar, Complete
	p.HandleCommand(&plugin.CommandOptions{Name: "Gobuild", NArgs: "*", Bang: true, Eval: "[getcwd(), expand('%:p')]"}, c.cmdBuild)
	p.HandleCommand(&plugin.CommandOptions{Name: "GoCover", Eval: "[getcwd(), expand('%:p')]"}, c.cmdCover)
	p.HandleCommand(&plugin.CommandOptions{Name: "Gofmt", Eval: "expand('%:p:h')"}, c.cmdFmt)
	p.HandleCommand(&plugin.CommandOptions{Name: "GoGenerateTest", NArgs: "*", Range: "%", Addr: "line", Bang: true, Eval: "expand('%:p:h')", Complete: "file"}, c.cmdGenerateTest)
	p.HandleFunction(&plugin.FunctionOptions{Name: "GoGuru", Eval: "[getcwd(), expand('%:p'), &modified, line2byte(line('.')) + (col('.')-2)]"}, c.funcGuru)
	p.HandleCommand(&plugin.CommandOptions{Name: "GoIferr", Eval: "expand('%:p')"}, c.cmdIferr)
	p.HandleCommand(&plugin.CommandOptions{Name: "Golint", NArgs: "?", Eval: "expand('%:p')", Complete: "customlist,GoLintCompletion"}, c.cmdLint)
	p.HandleCommand(&plugin.CommandOptions{Name: "Gometalinter", Eval: "getcwd()"}, c.cmdMetalinter)
	p.HandleCommand(&plugin.CommandOptions{Name: "Gorename", NArgs: "?", Bang: true, Eval: "[getcwd(), expand('%:p'), expand('<cword>')]"}, c.cmdRename)
	p.HandleCommand(&plugin.CommandOptions{Name: "Gorun", NArgs: "*", Eval: "expand('%:p')"}, c.cmdRun)
	p.HandleCommand(&plugin.CommandOptions{Name: "GorunLast", Eval: "expand('%:p')"}, c.cmdRunLast)
	p.HandleCommand(&plugin.CommandOptions{Name: "Gotest", NArgs: "*", Eval: "expand('%:p:h')"}, c.cmdTest)
	p.HandleCommand(&plugin.CommandOptions{Name: "GoSwitchTest", Eval: "[getcwd(), expand('%:p'), line2byte(line('.')) + (col('.')-2)]"}, c.cmdSwitchTest)
	p.HandleCommand(&plugin.CommandOptions{Name: "Govet", NArgs: "*", Eval: "[getcwd(), expand('%:p')]", Complete: "customlist,GoVetCompletion"}, c.cmdVet)

	// Commnad completion
	p.HandleFunction(&plugin.FunctionOptions{Name: "GoLintCompletion", Eval: "getcwd()"}, c.cmdLintComplete) // list the file, directory and go packages
	p.HandleFunction(&plugin.FunctionOptions{Name: "GoVetCompletion", Eval: "getcwd()"}, c.cmdVetComplete)   // flag for go tool vet

	// for debug
	p.HandleCommand(&plugin.CommandOptions{Name: "GoByteOffset", Eval: "expand('%:p')"}, c.cmdByteOffset)
	p.HandleCommand(&plugin.CommandOptions{Name: "GoBuffers"}, c.cmdBuffers)
	p.HandleCommand(&plugin.CommandOptions{Name: "GoWindows"}, c.cmdWindows)
	p.HandleCommand(&plugin.CommandOptions{Name: "GoTabpages"}, c.cmdTabpagas)

	delve.Register(ctx, p, buildctxt)

	return c
}
