//go:build windows

package main

const (
	tituloJanela = "PDF UTIL"
	versaoApp    = "1.0.0"
	autorApp     = "Renato Carvalhaes"
)

func linhaCredito() string {
	return "versão " + versaoApp + "  ·  Criado por " + autorApp
}
