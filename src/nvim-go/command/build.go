// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"nvim-go/config"
	"nvim-go/internal/log"
	"nvim-go/nvimutil"

	"github.com/pkg/errors"
)

// CmdBuildEval struct type for Eval of GoBuild command.
type CmdBuildEval struct {
	Cwd  string `msgpack:",array"`
	File string
}

func (c *Command) cmdBuild(bang bool, eval *CmdBuildEval) {
	ctx, cancel := context.WithCancel(cmdContext)
	c.TryCancel("Build", cancel)

	c.errs.Delete("Build")
	res := c.Build(ctx, config.BuildForce, eval)
	c.HandleError("Build", res)
}

// Build builds the current buffers package use compile tool that determined
// from the package directory structure.
func (c *Command) Build(ctx context.Context, bang bool, eval *CmdBuildEval) interface{} {
	defer nvimutil.Profile(time.Now(), "GoBuild")

	errch := make(chan interface{}, 1)
	go func() {
		if !bang {
			bang = config.BuildForce
		}

		cmd, err := c.compileCmd(ctx, bang, filepath.Dir(eval.File))
		if err != nil {
			errch <- errors.WithStack(err)
		}
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if buildErr := cmd.Run(); buildErr != nil {
			if _, ok := buildErr.(*exec.ExitError); ok {
				errlist, err := nvimutil.ParseError(stderr.Bytes(), eval.Cwd, &c.ctx.Build, nil)
				if err != nil {
					errch <- errors.WithStack(err)
				}
				errch <- errlist
			}
			errch <- errors.WithStack(buildErr)
		}
		errch <- nil
	}()

	select {
	case res := <-errch:
		if res != nil {
			return res
		}
		log.Debug("success")
		return nvimutil.EchoSuccess(c.Nvim, "GoBuild", fmt.Sprintf("compiler: %s", c.ctx.Build.Tool))
	case <-ctx.Done():
		log.Debug("cancel")
		<-errch
		return nil
	}
}

// compileCmd returns the *exec.Cmd corresponding to the compile tool.
func (c *Command) compileCmd(ctx context.Context, bang bool, dir string) (*exec.Cmd, error) {
	bin, err := exec.LookPath(c.ctx.Build.Tool)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	args := []string{}
	if len(config.BuildFlags) > 0 {
		args = append(args, config.BuildFlags...)
	}

	cmd := exec.CommandContext(ctx, bin, "build")
	cmd.Dir = dir

	switch c.ctx.Build.Tool {
	case "go":
		// Outputs the binary to DevNull if without bang
		if !bang && !isTest {
			args = append(args, "-o", os.DevNull)
		}
	case "gb":
		if !isTest {
			cmd.Dir = c.ctx.Build.ProjectRoot
		}
	}

	cmd.Args = append(cmd.Args, args...)

	return cmd, nil
}
