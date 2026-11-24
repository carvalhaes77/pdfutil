//go:build windows

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Job usa os mesmos campos do parametros.ini e da linha de comando.
type Job struct {
	Acao     string
	Saida    string
	Arquivos []string
	Modo       string
	Pagina     int
	Paginas    string
	Compressao int
}

func init() {
	api.DisableConfigDir()
}

func (j Job) normalizar() Job {
	j.Acao = strings.ToLower(strings.TrimSpace(j.Acao))
	j.Modo = strings.ToLower(strings.TrimSpace(j.Modo))
	j.Saida = strings.TrimSpace(j.Saida)
	j.Paginas = strings.TrimSpace(j.Paginas)
	var arqs []string
	for _, a := range j.Arquivos {
		a = strings.TrimSpace(a)
		if a != "" {
			arqs = append(arqs, a)
		}
	}
	j.Arquivos = arqs
	if j.Acao == "dividir" && j.Modo == "" {
		j.Modo = "paginas"
	}
	return j
}

func RunJob(job Job) ([]string, error) {
	job = job.normalizar()
	if err := validarJob(job); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(job.Saida, 0o755); err != nil {
		return nil, fmt.Errorf("não foi possível criar a pasta de saída: %w", err)
	}
	api.DisableConfigDir()
	switch job.Acao {
	case "juntar":
		return juntar(job)
	case "dividir":
		return dividir(job)
	case "comprimir":
		return comprimir(job)
	default:
		return nil, fmt.Errorf("ação desconhecida: %s", job.Acao)
	}
}

func validarJob(job Job) error {
	switch job.Acao {
	case "juntar", "dividir", "comprimir":
	default:
		return fmt.Errorf("informe acao=juntar, acao=dividir ou acao=comprimir")
	}
	if job.Saida == "" {
		return fmt.Errorf("informe a pasta de saída em saida=")
	}
	if len(job.Arquivos) == 0 {
		return fmt.Errorf("informe ao menos um PDF em arquivos=")
	}
	for _, a := range job.Arquivos {
		if !strings.EqualFold(filepath.Ext(a), ".pdf") {
			return fmt.Errorf("não é um PDF: %s", a)
		}
		st, err := os.Stat(a)
		if err != nil {
			return fmt.Errorf("não foi possível abrir %s: %w", a, err)
		}
		if st.IsDir() {
			return fmt.Errorf("é uma pasta, não um PDF: %s", a)
		}
	}
	switch job.Acao {
	case "juntar":
		if len(job.Arquivos) < 2 {
			return fmt.Errorf("para juntar, informe pelo menos dois PDF")
		}
	case "comprimir":
		if _, ok := nivelCompressao(job.Compressao); !ok {
			return fmt.Errorf("informe compressao=40, 55, 70 ou 85")
		}
	case "dividir":
		if len(job.Arquivos) != 1 {
			return fmt.Errorf("para dividir, informe um único PDF")
		}
		switch job.Modo {
		case "paginas":
		case "pagina":
			if job.Pagina < 1 {
				return fmt.Errorf("informe pagina= com um número a partir de 1")
			}
		case "intervalo":
			if job.Paginas == "" {
				return fmt.Errorf("informe paginas=, por exemplo 2-3")
			}
		default:
			return fmt.Errorf("modo de divisão desconhecido: use paginas, pagina ou intervalo")
		}
	}
	return nil
}

func confPDF() *model.Configuration {
	conf := model.NewStatelessConfiguration()
	conf.Offline = true
	conf.ValidationMode = model.ValidationRelaxed
	return conf
}

func reservar(caminho string) error {
	if _, err := os.Stat(caminho); err == nil {
		return fmt.Errorf("já existe e não será substituído: %s", caminho)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func nomeSaida(dir, arquivo, sufixo string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(arquivo), filepath.Ext(arquivo))
	dest := filepath.Join(dir, base+sufixo+".pdf")
	if err := reservar(dest); err != nil {
		return "", err
	}
	return dest, nil
}

func juntar(job Job) ([]string, error) {
	dest, err := nomeSaida(job.Saida, job.Arquivos[0], "_juntado")
	if err != nil {
		return nil, err
	}
	if err := api.MergeCreateFile(context.Background(), job.Arquivos, dest, false, confPDF()); err != nil {
		return nil, err
	}
	return []string{dest}, nil
}

func comprimir(job Job) ([]string, error) {
	var feitos []string
	for _, arq := range job.Arquivos {
		dest, err := comprimirArquivo(arq, job.Saida, job.Compressao)
		if err != nil {
			return feitos, err
		}
		feitos = append(feitos, dest)
	}
	return feitos, nil
}

func dividir(job Job) ([]string, error) {
	arq := job.Arquivos[0]
	switch job.Modo {
	case "pagina":
		dest, err := nomeSaida(job.Saida, arq, "_dividido")
		if err != nil {
			return nil, err
		}
		sel := []string{strconv.Itoa(job.Pagina)}
		if err := api.CollectFile(context.Background(), arq, dest, sel, confPDF()); err != nil {
			return nil, err
		}
		return []string{dest}, nil
	case "intervalo":
		dest, err := nomeSaida(job.Saida, arq, "_intervalo")
		if err != nil {
			return nil, err
		}
		if err := api.CollectFile(context.Background(), arq, dest, selecionarPaginas(job.Paginas), confPDF()); err != nil {
			return nil, err
		}
		return []string{dest}, nil
	default:
		return dividirPaginas(job.Saida, arq)
	}
}

func selecionarPaginas(texto string) []string {
	partes := strings.FieldsFunc(texto, func(r rune) bool {
		return r == ',' || r == ';'
	})
	var sel []string
	for _, p := range partes {
		p = strings.TrimSpace(p)
		if p != "" {
			sel = append(sel, p)
		}
	}
	return sel
}

func dividirPaginas(saida, arq string) ([]string, error) {
	n, err := api.PageCountFile(context.Background(), arq)
	if err != nil {
		return nil, err
	}
	if n < 1 {
		return nil, fmt.Errorf("o PDF não tem páginas")
	}
	destinos := make([]string, n)
	for i := 0; i < n; i++ {
		sufixo := "_dividido"
		if i > 0 {
			sufixo = fmt.Sprintf("_dividido%d", i+1)
		}
		destinos[i], err = nomeSaida(saida, arq, sufixo)
		if err != nil {
			return nil, err
		}
	}

	tmp, err := os.MkdirTemp("", "pdfutil-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	if err := api.SplitFile(context.Background(), arq, tmp, 1, confPDF()); err != nil {
		return nil, err
	}
	gerados, err := listarPaginas(tmp)
	if err != nil {
		return nil, err
	}
	if len(gerados) != n {
		return nil, fmt.Errorf("a divisão gerou %d arquivo(s) e o PDF tem %d página(s)", len(gerados), n)
	}
	for i, origem := range gerados {
		if err := moverArquivo(origem, destinos[i]); err != nil {
			return destinos[:i], err
		}
	}
	return destinos, nil
}

func listarPaginas(dir string) ([]string, error) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	type item struct {
		n    int
		path string
	}
	var itens []item
	for _, e := range entradas {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".pdf") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		i := strings.LastIndex(base, "_")
		if i < 0 {
			continue
		}
		num := base[i+1:]
		if hifen := strings.IndexByte(num, '-'); hifen >= 0 {
			num = num[:hifen]
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			continue
		}
		itens = append(itens, item{n: n, path: filepath.Join(dir, e.Name())})
	}
	sort.Slice(itens, func(i, j int) bool { return itens[i].n < itens[j].n })
	caminhos := make([]string, len(itens))
	for i, it := range itens {
		caminhos[i] = it.path
	}
	return caminhos, nil
}

func moverArquivo(origem, destino string) error {
	if err := os.Rename(origem, destino); err == nil {
		return nil
	}
	in, err := os.Open(origem)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destino, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(destino)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(destino)
		return closeErr
	}
	return os.Remove(origem)
}
