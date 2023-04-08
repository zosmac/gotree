// Copyright © 2022 The Gomon Project.

package main

/*
#include <libproc.h>
#include <sys/sysctl.h>
*/
import "C"

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"github.com/zosmac/gocore"
)

type (
	// Pid is the type for the process identifier.
	Pid int

	// Pids holds a list of pids.
	Pids []Pid

	// processTable defines a process table as a map of pids to processes.
	processTable map[Pid]*process

	// processTree organizes the process into a hierarchy
	processTree map[Pid]processTree

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

	var pids Pids
	for _, pid := range flags.pids {
		if _, ok := tb[pid]; ok {
			pids = append(pids, pid)
		}
	}
	if len(pids) == 0 {
		pids = Pids{1}
	}

	trx := processTree{}
	for _, pid := range pids {
		if len(findTree(trx, pid)) > 0 {
			continue // pid already found
		}
		tr := findTree(tr, pid)
		for pid := tb[pid].Ppid; pid > 0; pid = tb[pid].Ppid { // ancestors
			tr = processTree{pid: tr}
		}

		// insert each process' tree into the main tree
		tx := trx
	loop:
		for {
			// descend the main tree until the place for the subtree is found
			// each subtree has one top node, so this loop actually only has one iteration
			if len(tr) > 1 {
				panic(fmt.Sprintf("len %d, %#v", len(tr), tr))
			}
			for pid, t := range tr {
				if ty, ok := tx[pid]; !ok {
					tx[pid] = t
					break loop
				} else { // descend along the common branch
					tr = t
					tx = ty
				}
			}
		}
	}
	flatTree(tb, trx)

	return nil
}

// getPids gets the list of active processes by pid.
func getPids() (Pids, error) {
	n, err := C.proc_listpids(C.PROC_ALL_PIDS, 0, nil, 0)
	if n <= 0 {
		return nil, gocore.Error("proc_listpids", err)
	}

	var pid C.int
	buf := make([]C.int, n/C.int(unsafe.Sizeof(pid))+10)
	if n, err = C.proc_listpids(C.PROC_ALL_PIDS, 0, unsafe.Pointer(&buf[0]), n); n <= 0 {
		return nil, gocore.Error("proc_listpids", err)
	}
	n /= C.int(unsafe.Sizeof(pid))
	if int(n) < len(buf) {
		buf = buf[:n]
	}

	pids := make(Pids, len(buf))
	for i, pid := range buf {
		pids[int(n)-i-1] = Pid(pid) // Darwin returns pids in descending order, so reverse the order
	}
	return pids, nil
}

// buildTable builds a process table and captures current process state.
func buildTable() processTable {
	pids, err := getPids()
	if err != nil {
		panic(fmt.Errorf("could not build process table %v", err))
	}

	tb := make(map[Pid]*process, len(pids))
	for _, pid := range pids {
		var bsi C.struct_proc_bsdshortinfo
		if n := C.proc_pidinfo(
			C.int(pid),
			C.PROC_PIDT_SHORTBSDINFO,
			0,
			unsafe.Pointer(&bsi),
			C.int(C.PROC_PIDT_SHORTBSDINFO_SIZE),
		); n != C.int(C.PROC_PIDT_SHORTBSDINFO_SIZE) {
			continue
		}

		tb[pid] = &process{
			Pid:         pid,
			Ppid:        Pid(bsi.pbsi_ppid),
			CommandLine: pid.commandLine(),
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
		addPid(tr, pids)
	}

	return tr
}

// addPid adds a process into the tree.
func addPid(tr processTree, pids Pids) {
	if len(pids) == 0 {
		return
	}
	if _, ok := tr[pids[0]]; !ok {
		tr[pids[0]] = processTree{}
	}
	addPid(tr[pids[0]], pids[1:])
}

// flatTree starts the display of the process tree.
func flatTree(tb processTable, tr processTree) Pids {
	return flatTreeIndent(tb, tr, 0)
}

// flatTreeIndent recurses through the process tree to display hierarchy of processes.
func flatTreeIndent(tb processTable, tr processTree, indent int) Pids {
	if len(tr) == 0 {
		return nil
	}
	var flat Pids

	var pids Pids
	for pid := range tr {
		pids = append(pids, pid)
	}

	sort.Slice(pids, func(i, j int) bool {
		dti := depthTree(tr[pids[i]])
		dtj := depthTree(tr[pids[j]])
		return dti > dtj ||
			dti == dtj && pids[i] < pids[j]
	})

	for _, pid := range pids {
		flat = append(flat, pid)
		display(tb[pid], indent)
		flat = append(flat, flatTreeIndent(tb, tr[pid], indent+1)...)
	}

	return flat
}

// display shows the pid, command, arguments, and environment variables for a process.
func display(p *process, indent int) {
	tab := strings.Repeat("|\t", indent)
	bg := 40 // black background for pid
	for _, pid := range flags.pids {
		if pid == p.Pid {
			bg = 42 // green background for pid
		}
	}
	pid := fmt.Sprintf("%s\033[97;%2dm%7d", tab, bg, p.Pid)
	tab += "|\t"
	var cmd, args, envs string
	if len(p.Args) > 0 {
		cmd = p.Args[0]
	}
	if len(p.Args) > 1 {
		guide := "\033[m\n" + tab + "\033[34m"
		args = guide + strings.Join(p.Args[1:], guide)
	}
	if len(p.Envs) > 0 {
		guide := "\033[m\n" + tab + "\033[35m"
		envs = guide + strings.Join(p.Envs, guide)
	}
	fmt.Printf("%s\033[m %s%s%s\033[m\n", pid, cmd, args, envs)
}

// depthTree enables sort of deepest process trees first.
func depthTree(tr processTree) int {
	depth := 0
	for _, tree := range tr {
		dt := depthTree(tree) + 1
		if depth < dt {
			depth = dt
		}
	}
	return depth
}

// findTree finds the process tree parented by a specific process.
func findTree(tr processTree, parent Pid) processTree {
	for pid, tr := range tr {
		if pid == parent {
			return processTree{parent: tr}
		}
		if tr = findTree(tr, parent); tr != nil {
			return tr
		}
	}

	return nil
}

// commandLine retrieves process command, arguments, and environment.
func (pid Pid) commandLine() CommandLine {
	size := C.size_t(C.ARG_MAX)
	buf := make([]byte, size)

	if rv := C.sysctl(
		&[]C.int{C.CTL_KERN, C.KERN_PROCARGS2, C.int(pid)}[0],
		3,
		unsafe.Pointer(&buf[0]),
		&size,
		unsafe.Pointer(nil),
		0,
	); rv != 0 {
		return CommandLine{}
	}

	l := int(*(*uint32)(unsafe.Pointer(&buf[0])))
	ss := bytes.FieldsFunc(buf[4:size], func(r rune) bool { return r == 0 })
	var executable string
	var args, envs []string
	for i, s := range ss {
		if i == 0 {
			executable = string(s)
		} else if i <= l {
			args = append(args, string(s))
		} else {
			envs = append(envs, string(s))
		}
	}

	return CommandLine{
		Executable: executable,
		Args:       args,
		Envs:       envs,
	}
}
