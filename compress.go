//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type ajusteImagem struct {
	dpi       float64
	qualidade int64
}

func nivelCompressao(pct int) (ajusteImagem, bool) {
	switch pct {
	case 40:
		return ajusteImagem{dpi: 150, qualidade: 68}, true
	case 55:
		return ajusteImagem{dpi: 120, qualidade: 50}, true
	case 70:
		return ajusteImagem{dpi: 96, qualidade: 34}, true
	case 85:
		return ajusteImagem{dpi: 72, qualidade: 18}, true
	default:
		return ajusteImagem{}, false
	}
}

func comprimirArquivo(origem, pasta string, pct int) (string, error) {
	nivel, ok := nivelCompressao(pct)
	if !ok {
		return "", fmt.Errorf("compressão inválida: %d", pct)
	}
	info, err := os.Stat(origem)
	if err != nil {
		return "", err
	}
	dest, err := nomeSaida(pasta, origem, fmt.Sprintf("_comprimido%d", pct))
	if err != nil {
		return "", err
	}
	ajuste := nivel
	var ultimo string
	for tentativa := 0; tentativa < 4; tentativa++ {
		caminho, err := rasterizarParaPDF(origem, ajuste)
		if err != nil {
			return "", err
		}
		ultimo = caminho
		gerado, err := os.Stat(caminho)
		if err != nil {
			os.Remove(caminho)
			return "", err
		}
		if gerado.Size() < info.Size() {
			if err := moverArquivo(caminho, dest); err != nil {
				os.Remove(caminho)
				return "", err
			}
			return dest, nil
		}
		os.Remove(caminho)
		ajuste.qualidade -= 12
		ajuste.dpi *= 0.82
		if ajuste.qualidade < 12 {
			ajuste.qualidade = 12
		}
		if ajuste.dpi < 60 {
			ajuste.dpi = 60
		}
	}
	if ultimo != "" {
		os.Remove(ultimo)
	}
	return "", fmt.Errorf("não foi possível reduzir %s no nível %d%%", filepath.Base(origem), pct)
}

func rasterizarParaPDF(origem string, ajuste ajusteImagem) (string, error) {
	tmp, err := os.MkdirTemp("", "pdfutil-img-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	if err := renderizarPaginas(origem, tmp, ajuste); err != nil {
		return "", err
	}
	jpg, err := listarJPEG(tmp)
	if err != nil {
		return "", err
	}
	dims, err := api.PageDimsFile(context.Background(), origem)
	if err != nil {
		return "", err
	}
	if len(jpg) != len(dims) {
		return "", fmt.Errorf("a leitura gerou %d página(s) e o PDF tem %d", len(jpg), len(dims))
	}

	paginas := make([]string, len(jpg))
	for i, img := range jpg {
		paginas[i] = filepath.Join(tmp, fmt.Sprintf("pagina-%d.pdf", i+1))
		if err := imagemParaPDF(img, dims[i], paginas[i]); err != nil {
			return "", err
		}
	}
	saida := filepath.Join(os.TempDir(), fmt.Sprintf("pdfutil-comp-%d.pdf", os.Getpid()))
	os.Remove(saida)
	if len(paginas) == 1 {
		if err := moverArquivo(paginas[0], saida); err != nil {
			return "", err
		}
		return saida, nil
	}
	if err := api.MergeCreateFile(context.Background(), paginas, saida, false, confPDF()); err != nil {
		return "", err
	}
	return saida, nil
}

func imagemParaPDF(jpg string, dim types.Dim, out string) error {
	imp := pdfcpu.DefaultImportConfig()
	imp.UserDim = true
	imp.PageSize = ""
	imp.PageDim = &types.Dim{Width: dim.Width, Height: dim.Height}
	imp.Pos = types.Full
	imp.Scale = 1
	imp.InpUnit = types.POINTS
	return api.ImportImagesFile(context.Background(), []string{jpg}, out, imp, confPDF())
}

func listarJPEG(dir string) ([]string, error) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var imgs []string
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ".jpg" {
			imgs = append(imgs, filepath.Join(dir, e.Name()))
		}
	}
	if len(imgs) == 0 {
		return nil, fmt.Errorf("nenhuma página foi convertida em imagem")
	}
	sort.Strings(imgs)
	return imgs, nil
}

func renderizarPaginas(pdf, dir string, ajuste ajusteImagem) error {
	script := filepath.Join(dir, "render.ps1")
	corpo := strings.ReplaceAll(scriptRender, "IAOP", "IAsyncOperation`1")
	if err := os.WriteFile(script, []byte("\uFEFF"+corpo), 0o644); err != nil {
		return err
	}
	cmd := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-File", script,
		pdf,
		dir,
		fmt.Sprintf("%.0f", ajuste.dpi),
		fmt.Sprintf("%d", ajuste.qualidade),
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	saida, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("não foi possível converter as páginas em imagem: %w (%s)", err, string(saida))
	}
	return nil
}

const scriptRender = `
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Runtime.WindowsRuntime
Add-Type -AssemblyName System.Drawing
$null = [Windows.Storage.StorageFile, Windows.Storage, ContentType = WindowsRuntime]
$null = [Windows.Data.Pdf.PdfDocument, Windows.Data.Pdf, ContentType = WindowsRuntime]
$null = [Windows.Storage.Streams.InMemoryRandomAccessStream, Windows.Storage.Streams, ContentType = WindowsRuntime]

function AwaitOp($op, $type) {
  $m = [System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and $_.IsGenericMethodDefinition -and $_.GetParameters().Count -eq 1 -and
    $_.GetParameters()[0].ParameterType.Name -eq 'IAOP'
  } | Select-Object -First 1
  $task = $m.MakeGenericMethod($type).Invoke($null, @($op))
  $task.Wait() | Out-Null
  if ($task.IsFaulted) { throw $task.Exception.InnerException }
  return $task.Result
}
function AwaitAct($act) {
  $m = [System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and -not $_.IsGenericMethod -and $_.GetParameters().Count -eq 1 -and
    $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncAction'
  } | Select-Object -First 1
  $task = $m.Invoke($null, @($act))
  $task.Wait() | Out-Null
  if ($task.IsFaulted) { throw $task.Exception.InnerException }
}

$pdf = $args[0]
$dir = $args[1]
$dpi = [double]$args[2]
$qualidade = [long]$args[3]
$file = AwaitOp ([Windows.Storage.StorageFile]::GetFileFromPathAsync($pdf)) ([Windows.Storage.StorageFile])
$doc = AwaitOp ([Windows.Data.Pdf.PdfDocument]::LoadFromFileAsync($file)) ([Windows.Data.Pdf.PdfDocument])
$codec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object { $_.MimeType -eq 'image/jpeg' }
for ($i = 0; $i -lt $doc.PageCount; $i++) {
  $page = $doc.GetPage($i)
  $px = [uint32][Math]::Max(200, [Math]::Min(4000, [Math]::Round($page.Size.Width / 96.0 * $dpi)))
  $stream = New-Object Windows.Storage.Streams.InMemoryRandomAccessStream
  $opts = New-Object Windows.Data.Pdf.PdfPageRenderOptions
  $opts.DestinationWidth = $px
  AwaitAct ($page.RenderToStreamAsync($stream, $opts))
  $stream.Seek(0) | Out-Null
  $tamanho = [uint32]$stream.Size
  $reader = New-Object Windows.Storage.Streams.DataReader($stream)
  AwaitOp ($reader.LoadAsync($tamanho)) ([uint32]) | Out-Null
  $buf = New-Object byte[] $tamanho
  $reader.ReadBytes($buf)
  $ms = New-Object IO.MemoryStream(,$buf)
  $img = [System.Drawing.Image]::FromStream($ms)
  $ep = New-Object System.Drawing.Imaging.EncoderParameters(1)
  $ep.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter([System.Drawing.Imaging.Encoder]::Quality, $qualidade)
  $nome = Join-Path $dir ('{0:D4}.jpg' -f ($i + 1))
  $img.Save($nome, $codec, $ep)
  $img.Dispose()
  $ms.Dispose()
  $reader.Dispose()
  $stream.Dispose()
  $page.Dispose()
}
`
