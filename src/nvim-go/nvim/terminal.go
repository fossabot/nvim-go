package nvim

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"nvim-go/config"

	"github.com/garyburd/neovim-go/vim"
)

var (
	tbuffer vim.Buffer
	twindow vim.Window
)

type Terminal struct {
	v      *vim.Vim
	cmd    []string
	mode   string
	Width  int64
	Height int64
}

func NewTerminal(vim *vim.Vim, command []string, mode string) *Terminal {
	return &Terminal{
		v:    vim,
		cmd:  command,
		mode: mode,
	}
}

func (t *Terminal) Run() error {
	var (
		b      vim.Buffer
		w      vim.Window
		pos    = config.TerminalPosition
		height = config.TerminalHeight
		width  = config.TerminalWidth
	)

	// Creates a new pipeline
	p := t.v.NewPipeline()
	p.CurrentBuffer(&b)
	p.CurrentWindow(&w)
	if err := p.Wait(); err != nil {
		return err
	}

	if twindow != 0 {
		p.SetCurrentWindow(twindow)
		p.SetBufferOption(tbuffer, "modified", false)
		p.Call("termopen", nil, strings.Join(t.cmd, " "))
		p.SetBufferOption(tbuffer, "modified", true)
	} else {
		// Set split window position. (defalut: botright)
		vcmd := pos + " "

		t.Height = height
		t.Width = width

		switch {
		case t.Height != int64(0) && t.mode == "split":
			vcmd += strconv.FormatInt(t.Height, 10)
		case t.Width != int64(0) && t.mode == "vsplit":
			vcmd += strconv.FormatInt(t.Width, 10)
		case strings.Index(t.mode, "split") == -1:
			return errors.New(fmt.Sprintf("%s mode is not supported", t.mode))
		}

		// Create terminal buffer and spawn command.
		vcmd += t.mode + " | terminal " + strings.Join(t.cmd, " ")
		p.Command(vcmd)

		// Get terminal buffer and windows information.
		p.CurrentBuffer(&tbuffer)
		p.CurrentWindow(&twindow)
		if err := p.Wait(); err != nil {
			return err
		}

		// Workaround for "autocmd BufEnter term://* startinsert"
		if config.TerminalStartInsert {
			p.Command("stopinsert")
		}

		p.SetBufferOption(tbuffer, "filetype", "terminal")
		p.SetBufferOption(tbuffer, "buftype", "nofile")
		p.SetBufferOption(tbuffer, "bufhidden", "delete")
		p.SetBufferOption(tbuffer, "buflisted", false)
		p.SetBufferOption(tbuffer, "swapfile", false)

		p.SetWindowOption(twindow, "list", false)
		p.SetWindowOption(twindow, "number", false)
		p.SetWindowOption(twindow, "relativenumber", false)
		p.SetWindowOption(twindow, "winfixheight", true)
	}

	// Set buffer name, filetype and options
	p.SetBufferName(tbuffer, "__GO_TERMINAL__")

	// Refocus coding buffer
	p.SetCurrentWindow(w)

	return p.Wait()
}