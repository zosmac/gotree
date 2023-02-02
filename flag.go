// Copyright © 2023 The Gomon Project.

package main

import (
	"github.com/zosmac/gocore"
)

// init initializes the command line flags.
func init() {
	gocore.Flags.CommandDescription = `The gotree command produces a tree listing of the processes running currently on the system.`
}
