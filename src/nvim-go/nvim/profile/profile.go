// Copyright 2016 Koichi Shiraishi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package profile

import (
	"log"
	"time"
)

// Profile measurement of the time it took to any func and output log file.
// Usage: defer nvim.Profile(time.Now(), "func name")
func Start(start time.Time, name string) {
	elapsed := time.Since(start).Seconds()
	log.Printf("%s: %fsec\n", name, elapsed)
}
