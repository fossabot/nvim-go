// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autocmd

import (
	"context"

	"nvim-go/command"
	"nvim-go/config"
	"nvim-go/nvimutil"
	"path/filepath"

	"github.com/neovim/go-client/nvim"
)

type bufWritePostEval struct {
	Cwd  string `msgpack:",array"`
	File string
}

func (a *Autocmd) bufWritePost(eval *bufWritePostEval) {
	ctx, cancel := context.WithCancel(autocmdContext)
	a.cmd.TryCancel("BufWritePost", cancel)
	res := a.BufWritePost(ctx, eval)
	a.cmd.HandleError("BufWritePost", res)
}

// BufWritePost run the 'autosave' commands on BufWritePost autocmd.
func (a *Autocmd) BufWritePost(ctx context.Context, eval *bufWritePostEval) interface{} {
	dir := filepath.Dir(eval.File)

	if config.BuildAutosave {
		a.errs.Delete("Build")
		res := a.cmd.Build(ctx, config.BuildForce, &command.CmdBuildEval{
			Cwd:  eval.Cwd,
			File: eval.File,
		})
		if res != nil {
			return res
		}
	}

	if config.GolintAutosave {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()

			a.errs.Delete("Lint")
			if res := a.cmd.Lint(ctx, nil, eval.File); res != nil {
				a.errs.Store("Lint", res)
			}
		}()
	}

	if config.GoVetAutosave {
		a.wg.Add(1)
		a.mu.Lock()
		go func() {
			defer func() {
				a.wg.Done()
				a.mu.Unlock()
			}()

			a.errs.Delete("Vet")
			err := a.cmd.Vet(nil, &command.CmdVetEval{
				Cwd:  eval.Cwd,
				File: eval.File,
			})
			switch e := err.(type) {
			case error:
				nvimutil.ErrorWrap(a.Nvim, e)
			case []*nvim.QuickfixError:
				a.errs.Store("Vet", e)
			}
		}()
	}

	if config.MetalinterAutosave {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.cmd.Metalinter(eval.Cwd)
		}()
	}

	if config.TestAutosave {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.cmd.Test(nil, dir)
		}()
	}

	a.wg.Wait()
	errlist := make(map[string][]*nvim.QuickfixError)
	a.errs.Range(func(ki, vi interface{}) bool {
		k, v := ki.(string), vi.([]*nvim.QuickfixError)
		errlist[k] = append(errlist[k], v...)
		return true
	})

	if len(errlist) > 0 {
		return nvimutil.ErrorList(a.Nvim, errlist, true)
	}

	return nvimutil.ClearErrorlist(a.Nvim, true)
}
