//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type janela struct {
	mw       *walk.MainWindow
	ni       *walk.NotifyIcon
	lista    *walk.ListBox
	arquivos []string
	saida    *walk.LineEdit
	estado   *walk.Label
	gerar    *walk.PushButton
	dividir  *walk.GroupBox
	juntar   *walk.RadioButton
	divRadio *walk.RadioButton
	comp     *walk.RadioButton
	cada     *walk.RadioButton
	uma      *walk.RadioButton
	intervalo  *walk.RadioButton
	compGrupo *walk.GroupBox
	c40       *walk.RadioButton
	c55       *walk.RadioButton
	c70       *walk.RadioButton
	c85       *walk.RadioButton
	pagina    *walk.LineEdit
	faixa      *walk.LineEdit
	pastaUsada string
	inicio     *walk.CheckBox
	acaoIni  *walk.Action
	ocupado  bool
	saindo   bool
	avisou   bool
	ajustando bool
}

func runGUI() error {
	ui := &janela{}
	if err := garantirParametros(); err != nil {
		return err
	}
	ui.pastaUsada = lerChave("pasta")
	cinza := walk.RGB(110, 110, 110)

	err := MainWindow{
		AssignTo: &ui.mw,
		Title:    tituloJanela,
		MinSize:  Size{Width: 720, Height: 560},
		Size:     Size{Width: 760, Height: 600},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "Arquivos"},
			ListBox{AssignTo: &ui.lista, MinSize: Size{Height: 180}},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					botaoArquivos("Adicionar PDF", "", ui.adicionarPDF),
					botaoArquivos("Adicionar pasta", "", ui.adicionarPasta),
					botaoArquivos("Remover", "Remove o arquivo selecionado da janela Arquivos. O PDF no disco não é apagado.", ui.remover),
					botaoArquivos("Limpar", "Remove todos os arquivos da janela Arquivos. Os PDF no disco não são apagados.", ui.limpar),
					botaoArquivos("Subir", "Sobe o arquivo selecionado na sequência da janela Arquivos.", func() { ui.mover(-1) }),
					botaoArquivos("Descer", "Desce o arquivo selecionado na sequência da janela Arquivos.", func() { ui.mover(1) }),
					HSpacer{},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					Label{Text: "Saída"},
					LineEdit{AssignTo: &ui.saida, Text: lerChave("saida")},
					PushButton{Text: "Procurar", OnClicked: ui.escolherSaida},
				},
			},
			GroupBox{
				Title:  "Operação",
				Layout: HBox{},
				Children: []Widget{
					RadioButton{AssignTo: &ui.juntar, Text: "Juntar", OnClicked: ui.atualizarModo},
					RadioButton{AssignTo: &ui.divRadio, Text: "Dividir", OnClicked: ui.atualizarModo},
					RadioButton{AssignTo: &ui.comp, Text: "Comprimir", OnClicked: ui.atualizarModo},
				},
			},
			GroupBox{
				AssignTo: &ui.compGrupo,
				Title:    "Comprimir — a página vira imagem e a qualidade cai",
				Layout:   VBox{},
				Children: []Widget{
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							RadioButton{AssignTo: &ui.c40, Text: "40%"},
							RadioButton{AssignTo: &ui.c55, Text: "55%"},
							RadioButton{AssignTo: &ui.c70, Text: "70%"},
							RadioButton{AssignTo: &ui.c85, Text: "85%"},
						},
					},
					Label{Text: "Quanto maior a porcentagem, menor o PDF. O resultado continua PDF. Confira a leitura antes de enviar."},
				},
			},
			GroupBox{
				AssignTo: &ui.dividir,
				Title:    "Dividir",
				Layout:   HBox{},
				Children: []Widget{
					RadioButton{AssignTo: &ui.cada, Text: "Cada página"},
					RadioButton{AssignTo: &ui.uma, Text: "Uma página"},
					LineEdit{AssignTo: &ui.pagina, Text: "1", MaxSize: Size{Width: 56}},
					RadioButton{AssignTo: &ui.intervalo, Text: "Intervalo"},
					LineEdit{AssignTo: &ui.faixa, Text: "1-2", MinSize: Size{Width: 80}},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo:    &ui.gerar,
						Text:        "GERAR",
						Font:        Font{Family: "Segoe UI", PointSize: 18, Bold: true},
						MinSize:     Size{Width: 176, Height: 48},
						MaxSize:     Size{Width: 176, Height: 48},
						ToolTipText: "Gera o PDF novo na pasta de saída, conforme a operação escolhida. O arquivo original não é alterado.",
						OnClicked:   ui.gerarArquivos,
					},
					HSpacer{},
				},
			},
			Label{AssignTo: &ui.estado, Text: "Pronto."},
			CheckBox{AssignTo: &ui.inicio, Text: "Iniciar com o Windows neste usuário"},
			Label{
				Text:      linhaCredito(),
				Font:      Font{Family: "Segoe UI", PointSize: 8},
				TextColor: cinza,
			},
		},
	}.Create()
	if err != nil {
		return err
	}

	ui.mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !ui.saindo {
			*canceled = true
			ui.mw.Hide()
			if !ui.avisou && ui.ni != nil {
				ui.avisou = true
				_ = ui.ni.ShowInfo(tituloJanela, "Continua na bandeja. Use o botão direito para sair.")
			}
		}
	})

	if err := ui.criarBandeja(); err != nil {
		return err
	}
	ui.juntar.SetChecked(true)
	ui.cada.SetChecked(true)
	ui.c55.SetChecked(true)
	ui.atualizarModo()
	ui.saida.SetCueBanner("Pasta onde os PDF novos serão gravados")
	ui.aplicarInicioDoArquivo()
	ui.inicio.CheckedChanged().Attach(func() {
		if ui.ajustando {
			return
		}
		ui.definirInicio(ui.inicio.Checked())
	})

	ui.mw.Run()
	return nil
}

func (ui *janela) criarBandeja() error {
	ni, err := walk.NewNotifyIcon(ui.mw)
	if err != nil {
		return err
	}
	ui.ni = ni
	if icone, err := walk.NewIconFromResourceId(1); err == nil {
		_ = ui.mw.SetIcon(icone)
		_ = ni.SetIcon(icone)
	}
	_ = ni.SetToolTip(tituloJanela)
	_ = ni.SetVisible(true)
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			ui.mostrar(0)
		}
	})

	abrir := ui.acao("Abrir", func() { ui.mostrar(0) })
	juntar := ui.acao("Juntar", func() { ui.mostrar(1) })
	dividir := ui.acao("Dividir", func() { ui.mostrar(2) })
	comprimir := ui.acao("Comprimir", func() { ui.mostrar(3) })
	inicio := walk.NewAction()
	_ = inicio.SetText("Iniciar com o Windows")
	_ = inicio.SetCheckable(true)
	_ = inicio.SetChecked(inicioAutomaticoLigado())
	inicio.Triggered().Attach(func() { ui.definirInicio(!inicioAutomaticoLigado()) })
	ui.acaoIni = inicio
	sair := ui.acao("Sair", ui.sair)

	menu := ni.ContextMenu().Actions()
	_ = menu.Add(abrir)
	_ = menu.Add(juntar)
	_ = menu.Add(dividir)
	_ = menu.Add(comprimir)
	_ = menu.Add(walk.NewSeparatorAction())
	_ = menu.Add(inicio)
	_ = menu.Add(walk.NewSeparatorAction())
	_ = menu.Add(sair)
	return nil
}

func (ui *janela) acao(texto string, fn func()) *walk.Action {
	a := walk.NewAction()
	_ = a.SetText(texto)
	a.Triggered().Attach(fn)
	return a
}

func (ui *janela) mostrar(modo int) {
	ui.mw.Show()
	switch modo {
	case 1:
		ui.juntar.SetChecked(true)
	case 2:
		ui.divRadio.SetChecked(true)
	case 3:
		ui.comp.SetChecked(true)
	}
	ui.atualizarModo()
	ui.mw.SetFocus()
}

func (ui *janela) sair() {
	ui.saindo = true
	if ui.ni != nil {
		_ = ui.ni.SetVisible(false)
		ui.ni.Dispose()
	}
	ui.mw.Close()
}

func (ui *janela) atualizarModo() {
	if ui.dividir == nil || ui.divRadio == nil {
		return
	}
	ui.dividir.SetEnabled(ui.divRadio.Checked())
	if ui.compGrupo != nil && ui.comp != nil {
		ui.compGrupo.SetEnabled(ui.comp.Checked())
	}
}

func (ui *janela) atualizarLista() {
	copia := append([]string(nil), ui.arquivos...)
	_ = ui.lista.SetModel(copia)
}

func (ui *janela) adicionar(caminhos []string) {
	vistos := map[string]bool{}
	for _, a := range ui.arquivos {
		vistos[strings.ToLower(a)] = true
	}
	for _, c := range caminhos {
		c = filepath.Clean(c)
		if vistos[strings.ToLower(c)] {
			continue
		}
		vistos[strings.ToLower(c)] = true
		ui.arquivos = append(ui.arquivos, c)
		if ui.pastaUsada == "" {
			ui.pastaUsada = filepath.Dir(c)
		}
	}
	if len(caminhos) > 0 {
		ui.pastaUsada = filepath.Dir(caminhos[0])
		_ = definirChave("pasta", ui.pastaUsada)
	}
	ui.atualizarLista()
	if ui.saida.Text() == "" && len(ui.arquivos) > 0 {
		_ = ui.saida.SetText(filepath.Dir(ui.arquivos[0]))
		ui.guardarSaida()
	}
}

func (ui *janela) adicionarPDF() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Escolher PDF"
	dlg.Filter = "Arquivos PDF (*.pdf)|*.pdf"
	if dir := ui.pastaInicial(); dir != "" {
		dlg.InitialDirPath = dir
	}
	ok, err := dlg.ShowOpen(ui.mw)
	if err != nil || !ok {
		return
	}
	ui.adicionar([]string{dlg.FilePath})
}

func (ui *janela) adicionarPasta() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Escolher pasta com PDF"
	if dir := ui.pastaInicial(); dir != "" {
		dlg.InitialDirPath = dir
	}
	ok, err := dlg.ShowBrowseFolder(ui.mw)
	if err != nil || !ok {
		return
	}
	entradas, err := os.ReadDir(dlg.FilePath)
	if err != nil {
		ui.avisar(err.Error())
		return
	}
	var pdfs []string
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".pdf") {
			pdfs = append(pdfs, filepath.Join(dlg.FilePath, e.Name()))
		}
	}
	if len(pdfs) == 0 {
		ui.avisar("Não há PDF nesta pasta.")
		return
	}
	ui.adicionar(pdfs)
	ui.definirEstado(itoa(len(pdfs)) + " PDF da pasta.")
}

func botaoArquivos(texto, dica string, aoClicar walk.EventHandler) PushButton {
	return PushButton{
		Text:        texto,
		ToolTipText: dica,
		OnClicked:   aoClicar,
		Font:        Font{Family: "Segoe UI", PointSize: 7},
		MinSize:     Size{Width: 84, Height: 22},
		MaxSize:     Size{Width: 84, Height: 22},
	}
}

func (ui *janela) remover() {
	i := ui.lista.CurrentIndex()
	if i < 0 || i >= len(ui.arquivos) {
		return
	}
	ui.arquivos = append(ui.arquivos[:i], ui.arquivos[i+1:]...)
	ui.atualizarLista()
	if i >= len(ui.arquivos) {
		i = len(ui.arquivos) - 1
	}
	if i >= 0 {
		_ = ui.lista.SetCurrentIndex(i)
	}
}

func (ui *janela) limpar() {
	if len(ui.arquivos) == 0 {
		return
	}
	ui.arquivos = nil
	ui.atualizarLista()
}

func (ui *janela) mover(delta int) {
	i := ui.lista.CurrentIndex()
	j := i + delta
	if i < 0 || j < 0 || j >= len(ui.arquivos) {
		return
	}
	ui.arquivos[i], ui.arquivos[j] = ui.arquivos[j], ui.arquivos[i]
	ui.atualizarLista()
	_ = ui.lista.SetCurrentIndex(j)
}

func (ui *janela) escolherSaida() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Escolher pasta de saída"
	if dir := strings.TrimSpace(ui.saida.Text()); dir != "" {
		dlg.InitialDirPath = dir
	}
	ok, err := dlg.ShowBrowseFolder(ui.mw)
	if err != nil || !ok {
		return
	}
	_ = ui.saida.SetText(dlg.FilePath)
	ui.guardarSaida()
}

func (ui *janela) pastaInicial() string {
	if ui.pastaUsada != "" {
		return ui.pastaUsada
	}
	if dir := strings.TrimSpace(ui.saida.Text()); dir != "" {
		return dir
	}
	if len(ui.arquivos) > 0 {
		return filepath.Dir(ui.arquivos[0])
	}
	return ""
}

func (ui *janela) gerarArquivos() {
	if ui.ocupado {
		return
	}
	job := Job{
		Saida:    strings.TrimSpace(ui.saida.Text()),
		Arquivos: append([]string(nil), ui.arquivos...),
	}
	switch {
	case ui.divRadio.Checked():
		job.Acao = "dividir"
		switch {
		case ui.uma.Checked():
			job.Modo = "pagina"
			n, err := parseInt(ui.pagina.Text())
			if err != nil {
				ui.avisar("Informe o número da página.")
				return
			}
			job.Pagina = n
		case ui.intervalo.Checked():
			job.Modo = "intervalo"
			job.Paginas = ui.faixa.Text()
		default:
			job.Modo = "paginas"
		}
	case ui.comp.Checked():
		job.Acao = "comprimir"
		switch {
		case ui.c40.Checked():
			job.Compressao = 40
		case ui.c70.Checked():
			job.Compressao = 70
		case ui.c85.Checked():
			job.Compressao = 85
		default:
			job.Compressao = 55
		}
	default:
		job.Acao = "juntar"
	}
	ui.guardarSaida()
	ui.ocupado = true
	ui.gerar.SetEnabled(false)
	ui.definirEstado("Gerando...")
	go func(job Job) {
		feitos, err := RunJob(job)
		ui.mw.Synchronize(func() {
			ui.ocupado = false
			ui.gerar.SetEnabled(true)
			if err != nil {
				ui.definirEstado(err.Error())
				ui.avisar(err.Error())
				return
			}
			texto := "Concluído: " + itoa(len(feitos)) + " arquivo(s)."
			if len(feitos) > 0 && len(feitos) <= 8 {
				texto = strings.Join(feitos, "\n")
			}
			ui.definirEstado("Concluído: " + itoa(len(feitos)) + " arquivo(s) em " + job.Saida)
			walk.MsgBox(ui.mw, tituloJanela, texto, walk.MsgBoxIconInformation)
		})
	}(job)
}

func (ui *janela) definirEstado(texto string) {
	_ = ui.estado.SetText(texto)
}

func (ui *janela) avisar(texto string) {
	walk.MsgBox(ui.mw, tituloJanela, texto, walk.MsgBoxIconWarning)
}

func (ui *janela) ligarInicio(ligado bool) {
	ui.ajustando = true
	ui.inicio.SetChecked(ligado)
	if ui.acaoIni != nil {
		_ = ui.acaoIni.SetChecked(ligado)
	}
	ui.ajustando = false
}

func (ui *janela) guardarSaida() {
	if err := definirChave("saida", strings.TrimSpace(ui.saida.Text())); err != nil {
		ui.definirEstado(err.Error())
	}
}

func (ui *janela) aplicarInicioDoArquivo() {
	switch strings.ToLower(lerChave("iniciar")) {
	case "sim", "s", "true", "1":
		if !inicioAutomaticoLigado() {
			_ = ligarInicioAutomatico()
		}
	default:
		if inicioAutomaticoLigado() {
			_ = desligarInicioAutomatico()
		}
	}
	ui.ligarInicio(inicioAutomaticoLigado())
}

func (ui *janela) definirInicio(ligado bool) {
	if ui.ajustando {
		return
	}
	var err error
	if ligado {
		err = ligarInicioAutomatico()
	} else {
		err = desligarInicioAutomatico()
	}
	valor := "nao"
	if inicioAutomaticoLigado() {
		valor = "sim"
	}
	if chaveErr := definirChave("iniciar", valor); chaveErr != nil && err == nil {
		err = chaveErr
	}
	if err != nil {
		ui.avisar(err.Error())
	}
	ui.ligarInicio(inicioAutomaticoLigado())
}

func parseInt(s string) (int, error) {
	n := 0
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, os.ErrInvalid
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, os.ErrInvalid
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
