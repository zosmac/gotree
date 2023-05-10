// Copyright © 2023 The Gomon Project.

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zosmac/gocore"
)

type (
	// flagpids, because []Pid cannot be a receiver type for flag.Value Set and String.
	flagpids []Pid
)

var (
	// flags defines the command line flags.
	flags = struct {
		verbose bool
		pids    flagpids
	}{}
)

// init initializes the command line flags.
func init() {
	gocore.Flags.CommandDescription = `The gotree command prints a tree listing of the processes running currently on the system.`

	gocore.Flags.Var(
		&flags.verbose,
		"verbose",
		"[-verbose]",
		"Include full command path, arguments, and environment variables for each process in the list",
	)

	gocore.Flags.Var(
		&flags.pids,
		"pids",
		"[-pids <pid>[,<pid>...]]",
		"Print process tree for specific processes selected with comma separated list `pid[,pid...]`",
	)
}

// Set is a flag.Value interface method to enable logLevel as a command line flag.
func (pids *flagpids) Set(arg string) error {
	*pids = []Pid{}
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
func (pids flagpids) String() string {
	var args []string
	for _, pid := range pids {
		args = append(args, pid.String())
	}
	return strings.Join(args, ",")
}
