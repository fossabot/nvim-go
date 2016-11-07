// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nvim

import vim "github.com/neovim/go-client/nvim"

// WindowContext represents a Neovim window context.
type WindowContext struct {
	vim.Window
}
