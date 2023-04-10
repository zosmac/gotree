// Copyright © 2022 The Gomon Project.

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zosmac/gocore"
)

type (
	// Pid is the type for the process identifier.
	Pid int

	// Pids holds a list of pids.
	Pids []Pid

	Table[K ~int | ~string, V any] map[K]V

	// processTable defines a process table as a map of pids to processes.
	processTable = Table[Pid, *process]

	// Tree defines a hierarchy of objects of comparable type.
	Tree[K ~int | ~string] map[K]Tree[K]

	// processTree organizes the process into a hierarchy
	processTree = Tree[Pid]

	// process info.
	process struct {
		Pid
		Ppid Pid
		*CommandLine
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

	var pids Pids
	for _, pid := range flags.pids {
		if _, ok := tb[pid]; ok {
			pids = append(pids, pid)
		}
	}
	if len(pids) == 0 {
		pids = Pids{1}
	}

	tra := processTree{}
	for _, pid := range pids {
		if len(tra.findTree(pid)) > 0 {
			continue // pid already found
		}
		trb := tr.findTree(pid)
		for pid := tb[pid].Ppid; pid > 0; pid = tb[pid].Ppid { // ancestors
			trb = processTree{pid: trb}
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
	tra.flatTree(0, func(indent int, pid Pid) {
		display(indent, tb[pid])
	})

	return nil
}

// buildTable builds a process table and captures current process state.
func buildTable() processTable {
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
func buildTree(tb processTable) processTree {
	tr := processTree{}
	for pid := range tb {
		var pids Pids
		for ; pid > 0; pid = tb[pid].Ppid {
			pids = append(Pids{pid}, pids...)
		}
		tr.add(pids...)
	}

	return tr
}

// add inserts a node into a tree.
func (tr Tree[K]) add(nodes ...K) {
	if len(nodes) > 0 {
		if _, ok := tr[nodes[0]]; !ok {
			tr[nodes[0]] = Tree[K]{}
		}
		tr[nodes[0]].add(nodes[1:]...)
	}
}

// flatTreeIndent recurses through the tree to display as a hierarchy.
func (tr Tree[K]) flatTree(indent int, fn func(int, K)) []K {
	if len(tr) == 0 {
		return nil
	}
	var flat []K

	var keys []K
	for key := range tr {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		dti := tr[keys[i]].depthTree()
		dtj := tr[keys[j]].depthTree()
		return dti > dtj ||
			dti == dtj && keys[i] < keys[j]
	})

	for _, key := range keys {
		flat = append(flat, key)
		fn(indent, key)
		flat = append(flat, tr[key].flatTree(indent+1, fn)...)
	}

	return flat
}

// display shows the pid, command, arguments, and environment variables for a process.
func display(indent int, p *process) {
	tab := strings.Repeat("|\t", indent)
	var pid string
	if flags.verbose {
		bg := 44 // blue background for pid
		for _, pid := range flags.pids {
			if pid == p.Pid {
				bg = 41 // red background for pid
			}
		}
		pid = fmt.Sprintf("%s\033[97;%2dm%7d", tab, bg, p.Pid)
	} else {
		pid = fmt.Sprintf("%s%7d", tab, p.Pid)
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
	fmt.Printf("%s\033[m %s%s%s\033[m\n", pid, cmd, args, envs)
}

// depthTree enables sort of deepest process trees first.
func (tr Tree[K]) depthTree() int {
	depth := 0
	for _, tree := range tr {
		dt := tree.depthTree() + 1
		if depth < dt {
			depth = dt
		}
	}
	return depth
}

// findTree finds the subtree anchored by a specific node.
func (tr Tree[K]) findTree(node K) Tree[K] {
	for key, tr := range tr {
		if key == node {
			return Tree[K]{node: tr}
		}
		if tr = tr.findTree(node); tr != nil {
			return tr
		}
	}

	return nil
}
