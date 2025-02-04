// Copyright © 2022 The Gomon Project.

package main

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zosmac/gocore"
)

type (
	// Pid is the type for the process identifier.
	Pid int

	// process info.
	process struct {
		Pid
		Ppid Pid
		CommandLine
	}

	// CommandLine contains a process' command line arguments.
	CommandLine struct {
		Executable string   `json:"executable" gomon:"property"`
		Args       []string `json:"args" gomon:"property"`
		Envs       []string `json:"envs" gomon:"property"`
	}

	// tree alias organizes the process pids into a hierarchy.
	tree = gocore.Tree[Pid]

	// table alias defines a process table as a map of pids to processes.
	table = gocore.Table[Pid, *process]
)

// String renders the pid.
func (pid Pid) String() string {
	return strconv.Itoa(int(pid))
}

// main
func main() {
	gocore.Main(Main)
}

// Main builds and displays the process tree.
func Main(ctx context.Context) error {
	tb := buildTable()
	tr := buildTree(tb)

	var pids []Pid
	for _, pid := range flags.pids {
		if _, ok := tb[pid]; ok {
			pids = append(pids, pid)
		}
	}
	if len(pids) > 0 {
		pt := table{}
		for _, pid := range pids {
			for _, pid := range tr.Family(pid).All() {
				pt[pid] = tb[pid]
			}
		}
		tr = buildTree(pt)
	}

	for depth, pid := range tr.SortedFunc(func(a, b Pid) int {
		return cmp.Or(
			cmp.Compare(filepath.Base(tb[a].Executable), filepath.Base(tb[b].Executable)),
			cmp.Compare(a, b),
		)
	}) {
		display(depth, pid, tb[pid])
	}

	return nil
}

// buildTable builds a process table and captures current process state.
func buildTable() table {
	pids, err := getPids()
	if err != nil {
		panic(fmt.Errorf("could not build process table %v", err))
	}

	tb := make(map[Pid]*process, len(pids))
	for _, pid := range pids {
		if p := pid.process(); p != nil {
			tb[pid] = p
		}
	}

	return tb
}

// buildTree builds the process tree.
func buildTree(tb table) tree {
	tr := tree{}
	for pid := range tb {
		var pids []Pid
		for ; pid > 0; pid = tb[pid].Ppid {
			pids = append([]Pid{pid}, pids...)
		}
		tr.Add(pids...)
	}
	return tr
}

// display shows the pid, command, arguments, and environment variables for a process.
func display(indent int, _ Pid, p *process) {
	tab := strings.Repeat("|\t", indent)
	var s string
	if flags.verbose {
		bg := 44 // blue background for pid
		for _, pid := range flags.pids {
			if pid == p.Pid {
				bg = 41 // red background for pid
			}
		}
		s = fmt.Sprintf("%s\033[97;%2dm%7d", tab, bg, p.Pid)
	} else {
		s = fmt.Sprintf("%s%7d", tab, p.Pid)
	}
	tab += "|\t"

	var cmd, args, envs string
	if len(p.Args) > 0 {
		cmd = p.Args[0]
	}
	if flags.verbose {
		if len(p.Args) > 1 {
			guide := "\033[m\n" + tab + "\033[34m"
			args = guide + strings.Join(p.Args[1:], guide)
		}
		if len(p.Envs) > 0 {
			guide := "\033[m\n" + tab + "\033[35m"
			envs = guide + strings.Join(p.Envs, guide)
		}
	} else {
		cmd = filepath.Base(cmd)
	}
	fmt.Printf("%s\033[m %s%s%s\033[m\n", s, cmd, args, envs)
}
