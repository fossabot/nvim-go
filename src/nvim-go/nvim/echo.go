// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nvim

import (
	"fmt"
	"log"
	"strings"

	vim "github.com/neovim/go-client/nvim"
	"github.com/pkg/errors"
)

var (
	// ErrorColor highlight error message use Identifier syntax color.
	ErrorColor = "Identifier"
	// ProgressColor highlight progress message use Identifier syntax color.
	ProgressColor = "Identifier"
	// SuccessColor highlight success message use Identifier syntax color.
	SuccessColor = "Function"
)

// Echo provide the vim 'echo' command.
func Echo(v *vim.Nvim, format string, a ...interface{}) error {
	v.Command("redraw")
	return v.Command("echo '" + fmt.Sprintf(format, a...) + "'")
}

// EchoRaw provide the raw output vim 'echo' command.
func EchoRaw(v *vim.Nvim, a string) error {
	v.Command("redraw")
	return v.Command("echo \"" + a + "\"")
}

// Echomsg provide the vim 'echomsg' command.
func Echomsg(v *vim.Nvim, a ...interface{}) error {
	v.Command("redraw")
	return v.Command("echomsg '" + strings.TrimSpace(fmt.Sprintln(a...)) + "'")
}

// Echoerr provide the vim 'echoerr' command.
func Echoerr(v *vim.Nvim, format string, a ...interface{}) error {
	v.Command("redraw")
	return v.Command("echoerr '" + fmt.Sprintf(format, a...) + "'")
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// ErrorWrap splits the errors.Wrap's cause and error messages,
// and provide the vim 'echo' message with 'echohl' highlighting to cause text.
func ErrorWrap(v *vim.Nvim, err error) error {
	if err == nil {
		return nil
	}
	if IsDebug() {
		err, ok := err.(stackTracer)
		if ok {
			st := err.StackTrace()
			log.Printf("Error stack%+v", st[:])
		}
	}
	er := strings.SplitAfterN(err.Error(), ": ", 2)
	if strings.Contains(er[1], `"`) {
		er[1] = strings.Replace(er[1], `"`, "", -1)
	}
	v.Command("redraw")
	return v.Command("echo \"" + er[0] + "\" | echohl " + ErrorColor + " | echon \"" + er[1] + "\" | echohl None")
}

// EchohlErr provide the vim 'echo' command with the 'echohl' highlighting prefix text.
func EchohlErr(v *vim.Nvim, prefix string, a ...interface{}) error {
	v.Command("redraw")
	if prefix != "" {
		prefix += ": "
	}
	er := fmt.Sprintf("%s", a...)
	return v.Command("echo '" + prefix + "' | echohl " + ErrorColor + " | echon \"" + er + "\" | echohl None")
}

// EchohlBefore provide the vim 'echo' command with the 'echohl' highlighting prefix text.
func EchohlBefore(v *vim.Nvim, prefix string, highlight string, format string, a ...interface{}) error {
	v.Command("redraw")
	suffix := "\" | echohl None | echon \""
	if prefix != "" {
		suffix += ": "
	}
	return v.Command("echohl " + highlight + " | echo \"" + prefix + suffix + fmt.Sprintf(format, a...) + "\" | echohl None")
}

// EchohlAfter provide the vim 'echo' command with the 'echohl' highlighting message text.
func EchohlAfter(v *vim.Nvim, prefix string, highlight string, format string, a ...interface{}) error {
	v.Command("redraw")
	if prefix != "" {
		prefix += ": "
	}
	return v.Command("echo \"" + prefix + "\" | echohl " + highlight + " | echon \"" + fmt.Sprintf(format, a...) + "\" | echohl None")
}

// EchoProgress displays a command progress message to echo area.
func EchoProgress(v *vim.Nvim, prefix, format string, a ...interface{}) error {
	v.Command("redraw")
	msg := fmt.Sprintf(format, a...)
	return v.Command(fmt.Sprintf("echo \"%s: \" | echohl %s | echon \"%s ...\" | echohl None", prefix, ProgressColor, msg))
}

// EchoSuccess displays the success of the command to echo area.
func EchoSuccess(v *vim.Nvim, prefix string, msg string) error {
	v.Command("redraw")
	if msg != "" {
		msg = " | " + msg
	}
	return v.Command(fmt.Sprintf("echo \"%s: \" | echohl %s | echon 'SUCCESS' | echohl None | echon '%s' | echohl None", prefix, SuccessColor, msg))
}

// ReportError output of the accumulated errors report.
// TODO(zchee): research vim.ReportError behavior
// Why it does not immediately display error?
// func ReportError(v *vim.Nvim, format string, a ...interface{}) error {
// 	return v.ReportError(fmt.Sprintf(format, a...))
// }

// ClearMsg cleanups the echo area.
func ClearMsg(v *vim.Nvim) error {
	return v.Command("echon")
}
