# Go SSH Collector

Um script em Go para coletar dados de múltiplos dispositivos via SSH de forma concorrente e salvar os resultados em um arquivo Excel.

Este script é projetado para ser rápido e modular, permitindo que você altere facilmente o comando SSH e o dado que deseja extrair (via RegEx) sem precisar alterar a lógica principal.

## Funcionalidades

  * **Concorrente:** Usa goroutines para se conectar a dezenas ou centenas de dispositivos simultaneamente, tornando o processo de coleta extremamente rápido.
  * **Baseado em Excel:** Lê a lista de IPs de um arquivo `.xlsx` e salva os resultados de volta no mesmo arquivo.
  * **Mult-Planilha:** Processa automaticamente *todas* as planilhas (abas) presentes no arquivo Excel.
  * **Modular:** O comando, o padrão RegEx para captura e as colunas do Excel são facilmente configuráveis através de constantes no topo do código.

## Pré-requisitos

  * [Go](https://go.dev/doc/install) (versão 1.18+ recomendada)
  * Um arquivo Excel (por padrão `hosts.xlsx`) no mesmo diretório do script.
  * A estrutura esperada do Excel é:
      * Uma coluna para os IPs (configurada em `IPColumn`, padrão "B").
      * Uma coluna para o resultado (configurada em `ResultColumn`, padrão "C").
      * A primeira linha de cada planilha é tratada como cabeçalho e será ignorada.

## Configuração

Toda a configuração é feita no bloco `const` no início do arquivo `main.go`.

```go
// -------------------------------------------------------------------
// ALTERE ESTAS CONSTANTES PARA CUSTOMIZAR O SCRIPT
// -------------------------------------------------------------------
const (
	ExcelFile = "hosts.xlsx"
	Command   = "show device info"
	IPColumn  = "B" // Coluna com IPs
	ResultColumn = "C" // Coluna para salvar os resultados

	// RegexPattern: O padrão para buscar. O primeiro grupo de captura ()
	// é o dado que será extraído e salvo no Excel.
	RegexPattern = `Serial Number\s*:\s*(\S+)`

	// RegexError: A mensagem de erro para exibir se o padrão não for encontrado.
	RegexError = "serial number not found"
)
// -------------------------------------------------------------------
```

Por exemplo, se você quisesse capturar o "Uptime" do comando `show version`, você poderia alterar as constantes para:

```go
const (
    // ...
	Command   = "show version"
	ResultColumn = "D" // Salvar na coluna D
	RegexPattern = `Uptime is\s+(.*)`
	RegexError = "uptime not found"
    // ...
)
```

## Como Executar

1.  **Clone o repositório** (ou apenas salve o `main.go` em uma nova pasta).

2.  **Inicialize o módulo Go** (só precisa ser feito uma vez):

    ```bash
    go mod init meu-coletor
    ```

3.  **Baixe as dependências** (o `go mod tidy` vai encontrar o `excelize` e o `crypto/ssh`):

    ```bash
    go mod tidy
    ```

4.  **Execute o script:**

    ```bash
    go run .
    ```

5.  Digite seu usuário e senha SSH quando solicitado. O script irá se conectar a todos os IPs e preencher o arquivo Excel.
