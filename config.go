//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
)

func caminhoParametros() string {
	return pastaExe() + `\parametros.ini`
}

func modeloParametros(iniciar string) string {
	return "# PDF UTIL. Este arquivo fica na mesma pasta do pdfutil.exe.\r\n" +
		"# Se for apagado, o programa cria outro na próxima abertura.\r\n" +
		"# Nada é gravado no Registro do Windows.\r\n" +
		"#\r\n" +
		"# saida  = pasta onde os PDF novos são gravados\r\n" +
		"# pasta  = última pasta de onde os PDF foram escolhidos\r\n" +
		"# iniciar = sim abre com o Windows neste usuário; nao deixa desligado\r\n" +
		"#\r\n" +
		"# Os mesmos nomes servem no prompt. Veja: pdfutil ajuda\r\n" +
		"\r\n" +
		"saida=\r\n" +
		"pasta=\r\n" +
		"iniciar=" + iniciar + "\r\n" +
		"\r\n" +
		"# acao=juntar\r\n" +
		"# arquivos=C:\\a.pdf|C:\\b.pdf\r\n" +
		"# modo=paginas\r\n" +
		"# pagina=1\r\n" +
		"# paginas=1-2\r\n" +
		"# compressao=40\r\n"
}

func garantirParametros() error {
	if _, err := os.Stat(caminhoParametros()); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	iniciar := "nao"
	if inicioAutomaticoLigado() {
		iniciar = "sim"
	}
	return os.WriteFile(caminhoParametros(), []byte(modeloParametros(iniciar)), 0o644)
}

func lerChave(chave string) string {
	b, err := os.ReadFile(caminhoParametros())
	if err != nil {
		return ""
	}
	for _, linha := range strings.Split(string(b), "\n") {
		texto := strings.TrimSpace(strings.TrimRight(linha, "\r"))
		if texto == "" || strings.HasPrefix(texto, "#") || strings.HasPrefix(texto, ";") {
			continue
		}
		k, v, ok := strings.Cut(texto, "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), chave) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func definirChave(chave, valor string) error {
	if err := garantirParametros(); err != nil {
		return fmt.Errorf("não foi possível gravar parametros.ini na pasta do programa: %w", err)
	}
	b, err := os.ReadFile(caminhoParametros())
	if err != nil {
		return err
	}
	texto := strings.ReplaceAll(string(b), "\r\n", "\n")
	linhas := strings.Split(texto, "\n")
	achou := false
	for i, linha := range linhas {
		cru := strings.TrimSpace(linha)
		if cru == "" || strings.HasPrefix(cru, "#") || strings.HasPrefix(cru, ";") {
			continue
		}
		k, _, ok := strings.Cut(cru, "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), chave) {
			linhas[i] = chave + "=" + valor
			achou = true
			break
		}
	}
	if !achou {
		if len(linhas) > 0 && linhas[len(linhas)-1] != "" {
			linhas = append(linhas, "")
		}
		linhas = append(linhas, chave+"="+valor)
	}
	saida := strings.Join(linhas, "\r\n")
	if !strings.HasSuffix(saida, "\r\n") {
		saida += "\r\n"
	}
	return os.WriteFile(caminhoParametros(), []byte(saida), 0o644)
}
