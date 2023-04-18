// Copyright © 2022 The Gomon Project.

package main

/*
#include <libproc.h>
#include <sys/sysctl.h>
*/
import "C"
import (
	"bytes"
	"unsafe"

	"github.com/zosmac/gocore"
)

// getPids gets the list of active processes by pid.
func getPids() ([]Pid, error) {
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

	pids := make([]Pid, len(buf))
	for i, pid := range buf {
		pids[int(n)-i-1] = Pid(pid) // Darwin returns pids in descending order, so reverse the order
	}
	return pids, nil
}

func (pid Pid) process() *process {
	var bsi C.struct_proc_bsdshortinfo
	if n := C.proc_pidinfo(
		C.int(pid),
		C.PROC_PIDT_SHORTBSDINFO,
		0,
		unsafe.Pointer(&bsi),
		C.int(C.PROC_PIDT_SHORTBSDINFO_SIZE),
	); n != C.int(C.PROC_PIDT_SHORTBSDINFO_SIZE) {
		return nil
	}

	return &process{
		Pid:         pid,
		Ppid:        Pid(bsi.pbsi_ppid),
		CommandLine: pid.commandLine(),
	}
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
