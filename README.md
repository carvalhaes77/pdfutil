# PDF UTIL

Programa local para Windows 10 e Windows 11 que junta, divide e comprime PDF. Não instala nada, não pede senha de administrador e não precisa de internet.

O arquivo original não é alterado. O resultado é outro PDF, na pasta que a pessoa escolhe.

O executável pronto está em [Releases](https://github.com/carvalhaes77/pdfutil/releases): baixe o `pdfutil.exe`. Não precisa do código para usar.

## Requisitos

- Windows 10 ou Windows 11, 64 bits
- Nenhuma biblioteca, Java ou outro programa para instalar
- Para só usar: o arquivo `pdfutil.exe`
- Para compilar: Go 1.26 ou superior

## Como usar

Abra o `pdfutil.exe`. A janela se chama **PDF UTIL**. Se ela for fechada, o programa continua na bandeja do Windows, ao lado do relógio. O clique abre a janela de novo. O botão direito nesse ícone oferece as mesmas funções e a opção **Sair**.

1. Escolha um PDF ou uma pasta com PDFs.
2. Para juntar mais de um arquivo, use **Subir** e **Descer** até a ordem desejada.
3. Em **Saída**, escolha a pasta onde o arquivo novo será gravado. Se ficar em branco, vale a pasta do primeiro PDF.
4. Escolha a operação e clique em **Gerar**.

| Operação | Arquivo gerado |
| --- | --- |
| Juntar | `nome_juntado.pdf` |
| Dividir cada página | `nome_dividido.pdf`, `nome_dividido2.pdf`, `nome_dividido3.pdf` |
| Dividir uma página | `nome_dividido.pdf` |
| Dividir um intervalo | `nome_intervalo.pdf` |
| Comprimir | `nome_comprimido40.pdf`, `nome_comprimido55.pdf`, `nome_comprimido70.pdf` ou `nome_comprimido85.pdf` |

Na compressão, a folha continua do tamanho em que foi digitalizada, seja A4, carta ou outro. A porcentagem maior deixa o PDF menor e a leitura menos nítida. Abra o resultado e confira se ainda dá para ler. O texto da página passa a ser imagem; o arquivo final continua sendo PDF.

As pastas usadas da última vez ficam em `parametros.ini`, na mesma pasta do programa. Se esse arquivo for apagado, o PDF UTIL cria outro na próxima abertura, com as pastas em branco. O modelo comentado está em `parametros.ini.exemplo`. Arquivos temporários vão para a pasta temporária do Windows e são descartados ao terminar.

Quem for só distribuir o programa pode copiar apenas o `pdfutil.exe`.

## Linha de comando

Sem parâmetros, abre a janela. Com parâmetros, executa e termina, sem abrir a interface. Os nomes são os mesmos do `parametros.ini`.

```text
pdfutil ajuda
pdfutil acao=juntar saida=C:\saida arquivos=C:\a.pdf|C:\b.pdf
pdfutil acao=dividir saida=C:\saida modo=paginas arquivos=C:\doc.pdf
pdfutil acao=dividir saida=C:\saida modo=pagina pagina=3 arquivos=C:\doc.pdf
pdfutil acao=dividir saida=C:\saida modo=intervalo paginas=2-3 arquivos=C:\doc.pdf
pdfutil acao=comprimir saida=C:\saida compressao=55 arquivos=C:\doc.pdf
pdfutil ini
pdfutil ini C:\pasta\parametros.ini
```

No prompt, separe os PDF com `|`. Dentro do `parametros.ini` pode usar `|` ou `;`. Se o arquivo de saída já existe, a operação para e avisa.

| Chave | Função |
| --- | --- |
| `acao` | `juntar`, `dividir` ou `comprimir` |
| `saida` | Pasta onde os PDF novos são gravados |
| `arquivos` | Um ou mais PDF |
| `modo` | `paginas`, `pagina` ou `intervalo` (só em dividir) |
| `pagina` | Número da página, a partir de 1 |
| `paginas` | Intervalo, por exemplo `2-3` |
| `compressao` | `40`, `55`, `70` ou `85` |
| `iniciar` | `sim` ou `nao`. Atalho do usuário para abrir com o Windows. Não grava no Registro |

## Privacidade

Não há compartilhamento de dados. O PDF UTIL não envia documento, nome de arquivo nem conteúdo para a internet, para nuvem ou para outro computador. Não há conta, não há cadastro e não há serviço online no meio do caminho. O arquivo é lido e gravado somente nas pastas escolhidas neste computador.

O programa não guarda uma cópia em outro lugar e não define por quanto tempo os arquivos devem ser conservados. Quem usa continua responsável pelos documentos.

| Pergunta | Resposta |
| --- | --- |
| O documento sai deste computador? | Não. |
| Alguma nuvem recebe o PDF? | Não. |
| O programa precisa de internet para funcionar? | Não. |
| É preciso instalar biblioteca, Java ou outro programa? | Não. |
| O documento original é substituído? | Não. |

## Informações técnicas

O PDF UTIL é escrito em Go. A leitura, a junção, a divisão e a montagem do PDF ficam compiladas dentro do `pdfutil.exe`. Não há DLL, instalador nem runtime para copiar junto.

A compressão usa somente o recurso de PDF já presente no Windows 10 e no Windows 11. Cada página vira uma imagem, na resolução da porcentagem escolhida, e essa imagem volta para um arquivo PDF. A folha mantém o tamanho original. Nada disso é transmitido. O programa não abre conexão para enviar o documento.

O `pdfutil.exe` não grava configuração no Registro do Windows. A preferência de pastas fica no `parametros.ini`, ao lado do executável. O trabalho temporário fica em `%TEMP%` e é apagado ao fim da operação. Duas janelas ao mesmo tempo não abrem; a linha de comando continua podendo rodar com a janela aberta.

| Porcentagem | Efeito |
| --- | --- |
| 40 | Menor redução, leitura mais próxima do original |
| 55 | Meio-termo |
| 70 | Arquivo menor |
| 85 | Maior redução, conferir se o texto ainda se lê |

## Compilar

Na pasta deste repositório, no Windows:

```text
go-winres make --arch amd64
go build -ldflags "-s -w -H windowsgui" -o pdfutil.exe
```

`go-winres` grava o ícone, o manifest e os dados da aba Detalhes a partir de `winres\winres.json`. O manifest pede execução como o usuário atual, sem elevação.

## O que cada arquivo faz

| Arquivo | Função |
| --- | --- |
| `main.go` | Abre a janela ou a linha de comando e garante o `parametros.ini` |
| `gui.go` | Janela, bandeja e atalho de inicialização |
| `cli.go` | Parâmetros e texto de ajuda |
| `ops.go` | Juntar e dividir |
| `compress.go` | Comprimir, devolvendo um PDF |
| `config.go` | Leitura e gravação do `parametros.ini` |
| `startup.go` | Atalho em Inicializar, só deste usuário |
| `console_windows.go` | Ajuda visível quando o programa é chamado pelo prompt |
| `version.go` | Nome, versão e crédito |
| `app.manifest` | Compatibilidade com o Windows, sem pedir administrador |
| `winres\winres.json` | Ícone e dados do executável |
| `parametros.ini.exemplo` | Modelo, com pastas em branco |

## Licença

MIT. Pode usar, copiar e alterar, desde que o aviso de autoria permaneça. O texto está em `LICENSE`.

## Autor

Renato Carvalhaes. Versão 1.0.0.
