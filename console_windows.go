//go:build windows

package main

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func ensureConsole() {
	h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err == nil {
		tipo, tipoErr := windows.GetFileType(h)
		if tipoErr == nil && (tipo == windows.FILE_TYPE_DISK || tipo == windows.FILE_TYPE_PIPE) {
			return
		}
	}
	k := syscall.NewLazyDLL("kernel32.dll")
	r, _, _ := k.NewProc("AttachConsole").Call(0xFFFFFFFF)
	if r == 0 {
		k.NewProc("AllocConsole").Call()
	}
	f, openErr := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if openErr != nil {
		return
	}
	os.Stdout = f
	os.Stderr = f
}
