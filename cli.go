//go:build windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func runCLI(args []string) int {
	ensureConsole()
	if len(args) == 0 || ehAjuda(args[0]) {
		fmt.Fprint(os.Stdout, textoAjuda())
		return 0
	}
	job, err := interpretar(args)
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		fmt.Fprintln(os.Stdout)
		fmt.Fprint(os.Stdout, textoAjuda())
		return 2
	}
	feitos, err := RunJob(job)
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		for _, f := range feitos {
			fmt.Fprintln(os.Stdout, f)
		}
		return 1
	}
	for _, f := range feitos {
		fmt.Fprintln(os.Stdout, f)
	}
	fmt.Fprintf(os.Stdout, "Concluído: %d arquivo(s).\n", len(feitos))
	return 0
}

func ehAjuda(arg string) bool {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "ajuda", "help", "/?", "-h", "--help", "/h", "/help", "-help", "--ajuda", "/ajuda":
		return true
	default:
		return false
	}
}

func interpretar(args []string) (Job, error) {
	if len(args) == 1 && strings.EqualFold(args[0], "ini") {
		return lerINI(caminhoParametros())
	}
	if len(args) == 2 && strings.EqualFold(args[0], "ini") {
		return lerINI(args[1])
	}
	if len(args) == 1 && strings.EqualFold(filepath.Ext(args[0]), ".ini") {
		return lerINI(args[0])
	}

	var job Job
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--ini=") {
			return lerINI(strings.TrimPrefix(a, "--ini="))
		}
		if a == "--ini" || a == "/ini" {
			if i+1 >= len(args) {
				return Job{}, fmt.Errorf("faltou o caminho depois de %s", a)
			}
			return lerINI(args[i+1])
		}
		chave, valor, ok := strings.Cut(a, "=")
		if !ok {
			return Job{}, fmt.Errorf("parâmetro não reconhecido: %s", a)
		}
		if err := aplicarCampo(&job, chave, valor); err != nil {
			return Job{}, err
		}
	}
	return job, nil
}

func aplicarCampo(job *Job, chave, valor string) error {
	switch strings.ToLower(strings.TrimSpace(chave)) {
	case "acao":
		job.Acao = valor
	case "saida":
		job.Saida = valor
	case "arquivos":
		job.Arquivos = append(job.Arquivos, separarArquivos(valor)...)
	case "modo":
		job.Modo = valor
	case "pagina":
		n, err := strconv.Atoi(strings.TrimSpace(valor))
		if err != nil {
			return fmt.Errorf("pagina= precisa ser um número")
		}
		job.Pagina = n
	case "paginas":
		job.Paginas = valor
	case "compressao":
		n, err := strconv.Atoi(strings.TrimSpace(valor))
		if err != nil {
			return fmt.Errorf("compressao= precisa ser 40, 55, 70 ou 85")
		}
		job.Compressao = n
	default:
		return fmt.Errorf("parâmetro desconhecido: %s", chave)
	}
	return nil
}

func separarArquivos(valor string) []string {
	partes := strings.FieldsFunc(valor, func(r rune) bool {
		return r == '|' || r == ';'
	})
	var out []string
	for _, p := range partes {
		p = strings.Trim(strings.TrimSpace(p), `"`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func lerINI(caminho string) (Job, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return Job{}, fmt.Errorf("não foi possível ler %s: %w", caminho, err)
	}
	defer f.Close()
	var job Job
	sc := bufio.NewScanner(f)
	linha := 0
	for sc.Scan() {
		linha++
		texto := strings.TrimSpace(sc.Text())
		if texto == "" || strings.HasPrefix(texto, "#") || strings.HasPrefix(texto, ";") {
			continue
		}
		chave, valor, ok := strings.Cut(texto, "=")
		if !ok {
			return Job{}, fmt.Errorf("%s:%d: linha sem sinal de igual", caminho, linha)
		}
		if err := aplicarCampo(&job, chave, valor); err != nil {
			return Job{}, fmt.Errorf("%s:%d: %w", caminho, linha, err)
		}
	}
	if err := sc.Err(); err != nil {
		return Job{}, err
	}
	return job, nil
}

func pastaExe() string {
	p, err := os.Executable()
	if err != nil {
		return "."
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Dir(p)
	}
	return filepath.Dir(abs)
}

func textoAjuda() string {
	return `PDF UTIL ` + versaoApp + `
Criado por ` + autorApp + `

Sem parâmetros, abre a janela. O duplo clique faz a mesma coisa.
Quem não usa o prompt pode ignorar o restante.

  pdfutil
  pdfutil ajuda

Os nomes são os mesmos do arquivo parametros.ini,
que fica na mesma pasta do pdfutil.exe.

  acao       juntar | dividir | comprimir
  saida      pasta onde os PDF novos serão gravados
  arquivos   um ou mais PDF, separados por |
  modo       paginas | pagina | intervalo    (só em dividir)
  pagina     número da página, a partir de 1 (modo pagina)
  paginas    intervalo, por exemplo 2-3      (modo intervalo)
  compressao 40 | 55 | 70 | 85            (só em comprimir)
             vira imagem dentro do PDF; quanto maior, menor o arquivo

Pelo prompt:
  pdfutil acao=juntar saida=C:\saida arquivos=C:\a.pdf|C:\b.pdf
  pdfutil acao=dividir saida=C:\saida modo=paginas arquivos=C:\doc.pdf
  pdfutil acao=dividir saida=C:\saida modo=pagina pagina=3 arquivos=C:\doc.pdf
  pdfutil acao=dividir saida=C:\saida modo=intervalo paginas=2-3 arquivos=C:\doc.pdf
  pdfutil acao=comprimir saida=C:\saida compressao=55 arquivos=C:\doc.pdf

Pelo parametros.ini que estiver na mesma pasta do pdfutil.exe:
  pdfutil ini

Ou indicando outro arquivo:
  pdfutil ini C:\pasta\parametros.ini

No prompt, o ponto e vírgula separa comandos. Por isso, na linha de
comando, separe os PDF com |. Dentro do parametros.ini pode usar | ou ;.

Os originais não são alterados. Se o arquivo de saída já existe,
a operação para e avisa.

Nomes gerados, na pasta saida:
  juntar      nomeDoPrimeiro_juntado.pdf
  cada página nome_dividido.pdf, nome_dividido2.pdf, nome_dividido3.pdf
  uma página  nome_dividido.pdf
  intervalo   nome_intervalo.pdf
  comprimir   nome_comprimido40.pdf, nome_comprimido55.pdf,
              nome_comprimido70.pdf ou nome_comprimido85.pdf

Exemplo de parametros.ini:

  acao=juntar
  saida=C:\saida
  arquivos=C:\a.pdf|C:\b.pdf

`
}

func encodeUTF16LE(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, r := range u {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return b
}

func contemUTF16(haystack []byte, needle string) bool {
	if !utf8.ValidString(needle) {
		return false
	}
	n := encodeUTF16LE(needle)
	if len(n) == 0 || len(haystack) < len(n) {
		return false
	}
	for i := 0; i+len(n) <= len(haystack); i++ {
		ok := true
		for j := 0; j < len(n); j++ {
			if haystack[i+j] != n[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
