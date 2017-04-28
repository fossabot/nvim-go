// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autocmd

import (
	"context"
	"path/filepath"

	"nvim-go/config"
)

type bufWritePreEval struct {
	Cwd  string `msgpack:",array"`
	File string
}

func (a *Autocmd) bufWritePre(eval *bufWritePreEval) {
	ctx, cancel := context.WithCancel(autocmdContext)
	a.cmd.TryCancel("BufWritePre", cancel)
	res := a.BufWritePre(ctx, eval)
	a.cmd.HandleError("BufWritePre", res)
}

// BufWritePre run the commands on BufWritePre autocmd.
func (a *Autocmd) BufWritePre(ctx context.Context, eval *bufWritePreEval) interface{} {
	dir := filepath.Dir(eval.File)

	// Iferr need execute before Fmt function because that function calls "noautocmd write"
	// Also do not use goroutine.
	if config.IferrAutosave {
		err := a.cmd.Iferr(eval.File)
		if err != nil {
			return err
		}
	}

	if config.FmtAutosave {
		a.errs.Delete("Fmt")
		res := a.cmd.Fmt(ctx, dir)
		if res != nil {
			return res
		}
	}

	return nil
}
