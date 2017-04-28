// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package command

import (
	"bytes"
	"context"
	"go/scanner"
	"time"

	"nvim-go/config"
	"nvim-go/internal/log"
	"nvim-go/nvimutil"

	"github.com/neovim/go-client/nvim"
	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

var importsOptions = imports.Options{
	AllErrors: true,
	Comments:  true,
	TabIndent: true,
	TabWidth:  8,
}

func (c *Command) cmdFmt(dir string) {
	ctx, cancel := context.WithCancel(cmdContext)
	c.TryCancel("Fmt", cancel)

	c.errs.Delete("Fmt")
	res := c.Fmt(ctx, dir)
	c.HandleError("Fmt", res)
}

// Fmt format to the current buffer source uses gofmt behavior.
func (c *Command) Fmt(ctx context.Context, dir string) interface{} {
	defer nvimutil.Profile(time.Now(), "GoFmt")

	b := nvim.Buffer(c.ctx.BufNr)
	in, err := c.Nvim.BufferLines(b, 0, -1, true)
	if err != nil {
		return errors.WithStack(err)
	}

	var out [][]byte
	errch := make(chan interface{}, 1)
	go func() {
		switch config.FmtMode {
		case "fmt":
			importsOptions.FormatOnly = true
		case "goimports":
			// nothing to do
		default:
			errch <- errors.New("invalid value of go#fmt#mode option")
		}

		buf, formatErr := imports.Process("", nvimutil.ToByteSlice(in), &importsOptions)
		if formatErr != nil {
			bufName, err := c.Nvim.BufferName(b)
			if err != nil {
				errch <- errors.WithStack(err)
			}

			var errlist []*nvim.QuickfixError
			if e, ok := formatErr.(scanner.Error); ok {
				errlist = append(errlist, &nvim.QuickfixError{
					FileName: bufName,
					LNum:     e.Pos.Line,
					Col:      e.Pos.Column,
					Text:     e.Msg,
				})
			} else if el, ok := formatErr.(scanner.ErrorList); ok {
				for _, e := range el {
					errlist = append(errlist, &nvim.QuickfixError{
						FileName: bufName,
						LNum:     e.Pos.Line,
						Col:      e.Pos.Column,
						Text:     e.Msg,
					})
				}
			}
			errch <- errlist
		}
		out = nvimutil.ToBufferLines(bytes.TrimSuffix(buf, []byte{'\n'}))
		errch <- nil
	}()

	select {
	case res := <-errch:
		if err, ok := res.(error); ok {
			return errors.WithStack(err)
		}
		log.Debug("success")
		if err := c.minUpdate(in, out); err != nil {
			return errors.WithStack(err)
		}
		// TODO(zchee): When executed Fmt(itself) function at autocmd BufWritePre, vim "write"
		// command will starting before the finish of the Fmt function because that function called
		// asynchronously uses goroutine.
		// However, "noautocmd" raises duplicate the filesystem events.
		// In the case of macOS fsevents:
		//  (FSE_STAT_CHANGED -> FSE_CHOWN -> FSE_CONTENT_MODIFIED) x2.
		// It will affect the watchdog system such as inotify-tools, fswatch or fsevents-tools.
		// We need to consider the alternative of BufWriteCmd or other an effective way.
		return c.Nvim.Command("noautocmd write")
	case <-ctx.Done():
		log.Debug("cancel")
		<-errch
		return nil
	}
}

func (c *Command) minUpdate(in [][]byte, out [][]byte) error {
	// Find matching head lines.
	n := len(out)
	if len(in) < len(out) {
		n = len(in)
	}
	head := 0
	for ; head < n; head++ {
		if !bytes.Equal(in[head], out[head]) {
			break
		}
	}

	// Nothing to do?
	if head == len(in) && head == len(out) {
		return nil
	}

	// Find matching tail lines.
	n -= head
	tail := 0
	for ; tail < n; tail++ {
		if !bytes.Equal(in[len(in)-tail-1], out[len(out)-tail-1]) {
			break
		}
	}

	// Update the buffer.
	start := head
	end := len(in) - tail
	replace := out[head : len(out)-tail]

	return c.Nvim.SetBufferLines(nvim.Buffer(c.ctx.BufNr), start, end, true, replace)
}
