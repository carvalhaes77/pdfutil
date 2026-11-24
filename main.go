//go:build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func main() {
	if err := garantirParametros(); err != nil {
		if len(os.Args) > 1 {
			ensureConsole()
			fmt.Fprintln(os.Stdout, err.Error())
		} else {
			caixa(tituloJanela, err.Error())
			return
		}
	}
	if len(os.Args) > 1 {
		os.Exit(runCLI(os.Args[1:]))
	}
	if jaEstaAberto() {
		caixa("PDF UTIL", "PDF UTIL já está aberto na bandeja.")
		return
	}
	if err := runGUI(); err != nil {
		caixa("PDF UTIL", err.Error())
		os.Exit(1)
	}
}

var mutexAberto windows.Handle

func jaEstaAberto() bool {
	nome, err := windows.UTF16PtrFromString("Local\\PDFUTIL")
	if err != nil {
		return false
	}
	h, err := windows.CreateMutex(nil, false, nome)
	if err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			windows.CloseHandle(h)
		}
		return true
	}
	mutexAberto = h
	return false
}

func caixa(titulo, texto string) {
	t, _ := windows.UTF16PtrFromString(titulo)
	m, _ := windows.UTF16PtrFromString(texto)
	windows.MessageBox(0, m, t, windows.MB_OK|windows.MB_ICONINFORMATION)
}
