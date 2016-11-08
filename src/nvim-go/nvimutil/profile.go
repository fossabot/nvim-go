// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nvimutil

import (
	"log"
	"os"
	"time"
)

// Profile measurement of the time it took to any func and output log file.
// Usage: defer nvim.Profile(time.Now(), "func name")
func Profile(start time.Time, name string) {
	elapsed := time.Since(start).Seconds()
	if os.Getenv("NVIM_GO_DEBUG") != "" {
		log.Printf("%s: %fsec\n", name, elapsed)
	}
}