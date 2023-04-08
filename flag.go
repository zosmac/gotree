// Copyright © 2023 The Gomon Project.

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zosmac/gocore"
)

var (
	// flags defines the command line flags.
	flags = struct {
		pids Pids
	}{}
)

// init initializes the command line flags.
func init() {
	gocore.Flags.CommandDescription = `The gotree command prints a tree listing of the processes running currently on the system.`

	gocore.Flags.Var(
		&flags.pids,
		"pids",
		"[-pids <pid>[,<pid>...]]",
		"Print process tree for specific processes selected with comma separated list of `pids`",
	)
}

// Set is a flag.Value interface method to enable logLevel as a command line flag.
func (pids *Pids) Set(arg string) error {
	*pids = Pids{}
	args := strings.Split(arg, ",")
	for _, arg := range args {
		pid, err := strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("%s, %v", arg, err)
		}
		*pids = append(*pids, Pid(pid))
	}
	return nil
}

// String is a flag.Value interface method to enable logLevel as a command line flag.
func (pids Pids) String() string {
	var args []string
	for _, pid := range pids {
		args = append(args, strconv.Itoa(int(pid)))
	}
	return strings.Join(args, ",")
}
