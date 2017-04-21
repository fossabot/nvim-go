// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"nvim-go/config"
	"nvim-go/nvimutil"

	"github.com/cweill/gotests/gotests/process"
	"github.com/davecgh/go-spew/spew"
	"github.com/neovim/go-client/nvim"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/ast/astutil"
)

var generateFuncRe = regexp.MustCompile(`(?m)^func\s(?:\(\w\s[[:graph:]]+\)\s)?([\w]+)\(`)

func (c *Commands) cmdGenerateTest(args []string, ranges [2]int, bang bool, dir string) {
	go c.GenerateTest(args, ranges, bang, dir)
}

// GenerateTest generates the test files based by current buffer or args files
// functions.
func (c *Commands) GenerateTest(args []string, ranges [2]int, bang bool, dir string) error {
	defer nvimutil.Profile(time.Now(), "GenerateTest")

	b := nvim.Buffer(c.ctx.BufNr)
	if len(args) == 0 {
		f, err := c.Nvim.BufferName(b)
		if err != nil {
			return nvimutil.ErrorWrap(c.Nvim, errors.WithStack(err))
		}
		args = []string{f}
	}

	opt := &process.Options{
		WriteOutput:   true,
		PrintInputs:   true,
		AllFuncs:      config.GenerateTestAllFuncs,
		ExclFuncs:     config.GenerateTestExclFuncs,
		ExportedFuncs: config.GenerateTestExportedFuncs,
		Subtests:      config.GenerateTestSubTest,
	}

	// Check users used range. range return variable: (1,$)
	// If not used visual range, always ranges[0] is 0.
	if ranges[0] != 1 {
		// Re-check range[1] is not buffer line count
		lines, err := c.Nvim.BufferLineCount(b)
		if err != nil {
			return nvimutil.ErrorWrap(c.Nvim, errors.WithStack(err))
		}

		if ranges[1] != lines {
			start, end := ranges[0], ranges[1]
			// Get the buffer 2D slice
			// Neovim range value is based 1
			blines, err := c.Nvim.BufferLines(b, start-1, end, true)
			if err != nil {
				return nvimutil.ErrorWrap(c.Nvim, errors.WithStack(err))
			}
			// Convert to 1D byte slice
			buf := nvimutil.ToByteSlice(blines)

			matches := generateFuncRe.FindAllSubmatch(buf, -1)
			var onlyFuncs []string
			for _, fnName := range matches {
				onlyFuncs = append(onlyFuncs, string(fnName[1]))
			}
			opt.AllFuncs = false
			opt.ExportedFuncs = false
			// Set onlyFuncs option
			// like "-only=^(fooFunc|barFunc)$"
			opt.OnlyFuncs = fmt.Sprintf("^(%s)$", strings.Join(onlyFuncs, "|"))
		}
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	process.Run(w, args, opt)

	w.Close()
	os.Stdout = oldStdout

	if !bang {
		var genFuncs string
		scan := bufio.NewScanner(r)
		for scan.Scan() {
			genFuncs += scan.Text() + "\n"
		}

		// TODO(zchee): More beautiful code
		suffix := "_test.go "
		var ftests, ftestsRel string
		for _, f := range args {
			fnAbs := strings.Split(f, filepath.Ext(f))
			ftests += fnAbs[0] + suffix

			_, fnRel := filepath.Split(fnAbs[0])
			ftestsRel += fnRel + suffix
		}

		ask := fmt.Sprintf("%s\nGoGenerateTest: Generated %s\nGoGenerateTest: Open it? (y, n): ", genFuncs, ftestsRel)
		var answer interface{}
		if err := c.Nvim.Call("input", &answer, ask); err != nil {
			return nvimutil.ErrorWrap(c.Nvim, errors.WithStack(err))
		}
		// TODO(zchee): Support open the ftests[0] file only.
		// If passes multiple files for 'edit' commands, occur 'E172: Only one file name allowed' errror.
		if answer.(string) == "y" {
			return c.Nvim.Command(fmt.Sprintf("edit %s", ftests))
		}
	}

	return nil
}

type generateDocEval struct {
	File string `msgpack:",array"`
}

func (c *Commands) cmdGenerateDoc(filename string) {
	go c.GenerateDoc(filename)
}

// GenerateDoc generates the godoc comment.
func (c *Commands) GenerateDoc(filename string) error {
	defer nvimutil.Profile(time.Now(), "GenerateDoc")

	b := nvim.Buffer(c.ctx.BufNr)
	w := nvim.Window(c.ctx.WinID)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nvimutil.ErrorWrap(c.Nvim, err)
	}
	offset, err := nvimutil.ByteOffset(c.Nvim, b, w)
	if err != nil {
		return nvimutil.ErrorWrap(c.Nvim, errors.WithStack(err))
	}
	nodes, _ := astutil.PathEnclosingInterval(f, token.Pos(offset), token.Pos(offset))

	var fnDecl *ast.FuncDecl
	for _, n := range nodes {
		if d, ok := n.(*ast.FuncDecl); ok {
			fnDecl = d
			break
		}
	}

	// if fnDecl.Doc != nil {
	// 	err := errors.New("already have godoc comments")
	// 	return nvimutil.ErrorWrap(c.Nvim, err)
	// }

	for i, decl := range f.Decls {
		if d, ok := decl.(*ast.FuncDecl); ok && d == fnDecl {
			// log.Printf("f.Decls[i]: %T => %+v\n", f.Decls[i].(*ast.FuncDecl).Doc.List[0], f.Decls[i].(*ast.FuncDecl).Doc.List[0])
			f.Decls[i].(*ast.FuncDecl).Doc = &ast.CommentGroup{
				List: []*ast.Comment{
					&ast.Comment{
						Slash: fnDecl.Type.Pos(),
						Text:  "// " + fnDecl.Name.Name + " \n",
					},
				},
			}
			log.Printf("f.Decls[i]: %T => %+v\n", f.Decls[i].(*ast.FuncDecl).Doc.List[0], f.Decls[i].(*ast.FuncDecl).Doc.List[0])
		}
	}
	var buf bytes.Buffer
	log.Printf("f.Comments:\n%+v\n", spew.Sdump(f.Comments))
	if err := format.Node(&buf, fset, f); err != nil {
		return nvimutil.ErrorWrap(c.Nvim, errors.WithStack(err))
	}

	return c.Nvim.SetBufferLines(b, 0, -1, true, nvimutil.ToBufferLines(buf.Bytes()))
}
