// Copyright © 2022 The Gomon Project.

package main

import (
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

	// table defines a process table as a map of pids to processes.
	table = gocore.Table[Pid, *process]

	// tree organizes the process into a hierarchy
	tree = gocore.Tree[Pid]

	// meta defines the metadata for the tree.
	meta = gocore.Meta[Pid, *process, string]

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
	if len(pids) == 0 {
		pids = []Pid{1}
	}

	tra := tree{}
	for _, pid := range pids {
		if len(tra.FindTree(pid)) > 0 {
			continue // pid already found
		}
		trb := tr.FindTree(pid)
		for pid := tb[pid].Ppid; pid > 0; pid = tb[pid].Ppid { // ancestors
			trb = tree{pid: trb}
		}

		// insert each process' tree into the main tree
		trc := tra
	loop:
		for {
			// descend the main tree until the place for the subtree is found
			// each subtree has one top node, so this loop actually only has one iteration
			if len(trb) > 1 {
				panic(fmt.Sprintf("len %d, %#v", len(trb), trb))
			}
			for pid, trd := range trb {
				if tre, ok := trc[pid]; !ok {
					trc[pid] = trd
					break loop
				} else { // descend along the common branch
					trb = trd
					trc = tre
				}
			}
		}
	}

	for depth, pid := range (meta{
		Tree:  tra,
		Table: tb,
		// Order: func(node Pid, _ *process) int {
		// 	return depthTree(tra.FindTree(node))
		Order: func(node Pid, p *process) string {
			if len(p.Args) == 0 {
				return "."
			}
			return filepath.Base(p.Args[0])
		}}).All() {
		display(depth, pid, tb[pid])
	}

	return nil
}

// depthTree enables sort of deepest process trees first.
func depthTree(tr tree) int {
	depth := 0
	for _, tr := range tr {
		depth = max(depth, depthTree(tr)+1)
	}
	return depth
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
