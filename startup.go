//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func caminhoExe() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(p)
}

func atalhoInicializar() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("a pasta do usuário não está disponível")
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "PDF UTIL.lnk"), nil
}

func inicioAutomaticoLigado() bool {
	exe, err := caminhoExe()
	if err != nil {
		return false
	}
	lnk, err := atalhoInicializar()
	if err != nil {
		return false
	}
	b, err := os.ReadFile(lnk)
	if err != nil {
		return false
	}
	return contemUTF16(b, exe)
}

func ligarInicioAutomatico() error {
	exe, err := caminhoExe()
	if err != nil {
		return err
	}
	lnk, err := atalhoInicializar()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(lnk), 0o755); err != nil {
		return err
	}
	script := fmt.Sprintf(`$shell = New-Object -ComObject WScript.Shell
$lnk = $shell.CreateShortcut('%s')
$lnk.TargetPath = '%s'
$lnk.WorkingDirectory = '%s'
$lnk.WindowStyle = 1
$lnk.Description = 'PDF UTIL'
$lnk.Save()
`, psQuote(lnk), psQuote(exe), psQuote(filepath.Dir(exe)))
	tmp, err := os.CreateTemp("", "pdfutil-atalho-*.ps1")
	if err != nil {
		return err
	}
	nome := tmp.Name()
	defer os.Remove(nome)
	if _, err := tmp.Write([]byte("\uFEFF" + script)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", nome)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	saida, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("não foi possível criar o atalho: %w (%s)", err, string(saida))
	}
	return nil
}

func desligarInicioAutomatico() error {
	lnk, err := atalhoInicializar()
	if err != nil {
		return err
	}
	if err := os.Remove(lnk); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func psQuote(s string) string {
	return stringsReplace(s, "'", "''")
}

func stringsReplace(s, old, novo string) string {
	out := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return out + s
		}
		out += s[:i] + novo
		s = s[i+len(old):]
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
