package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)


const (
	ExcelFile    = "WiFi_Access_Points_-_Instantaneous.xlsx"
	Command      = "show device info"
	IPColumn     = "B" // Coluna com os IPs
	ResultColumn = "C" // Coluna em que os resultados serão escritos

	// RegexPattern: O regex que vai ser procurado. O grupo entre parênteses contém os dados que serão colocados nas células
	RegexPattern = `Serial Number\s*:\s*(\S+)`

	// RegexError: A mensagem de erro se o padrão não for encontrado no output
	RegexError = "serial number not found"
)

// -------------------------------------------------------------------

// Esta estrutura conterá o resultado de cada tarefa paralela
type Result struct {
	IP   string
	Data string
	Err  error
}

func main() {
	// 1. SOLICITAR INPUT DE CREDENCIAIS
	fmt.Print("Username: ")
	reader := bufio.NewReader(os.Stdin)
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("Failed to read password: %v", err)
	}
	password := string(bytePassword)
	fmt.Println() // Add a newline after password entry

	// 2. COMPILAR REGEX
	re := regexp.MustCompile(RegexPattern)

	// 3. ABRIR EXCEL
	f, err := excelize.OpenFile(ExcelFile)
	if err != nil {
		log.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close() // Garante que o arquivo vai fechar quando a main() encerrar

	// Pega a list de todas as folhas de planilha em um arquivo
	sheetList := f.GetSheetList()

	// 4. ITERA SOBRE CADA FOLHA E A PROCESSA
	for _, sheetName := range sheetList {
		fmt.Printf("--- Processing sheet: %s ---\n", sheetName)

		// 4a. LER IPs DA FOLHA ATUAL
		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Printf("Failed to get rows from sheet %s: %v. Skipping.", sheetName, err)
			continue // Pula essa folha e vai para a próxima
		}

		var ips []string
		// Mantém a ordem de qual linha vai receber o devido valor depois
		ipToRowIndex := make(map[string]int)

		for i, row := range rows {
			if i == 0 { // Pula a linha de "cabeçalho"
				continue
			}
			// A coluna B é o index 1.
			if len(row) >= 2 {
				ip := row[1]
				if ip != "" {
					ips = append(ips, ip)
					ipToRowIndex[ip] = i + 1
				}
			}
		}

		if len(ips) == 0 {
			log.Printf("No IPs found in sheet %s. Skipping.", sheetName)
			continue // Pula a folha
		}

		// 4b. PROCESSA OS IPs COM PARALELISMO (para essa folha)
		var wg sync.WaitGroup
		resultsChannel := make(chan Result, len(ips))

		fmt.Printf("Starting SSH connections to %d hosts from sheet %s...\n", len(ips), sheetName)

		for _, ip := range ips {
			wg.Add(1)
			// Passa o regex compilado 're' para o worker
			go getDeviceData(ip, username, password, &wg, resultsChannel, re)
		}

		// Goroutine que espera todos os workers terminarem, e depois fecha o canal
		go func() {
			wg.Wait()
			close(resultsChannel)
		}()

		// 4c. COLETA OS RESULTADOS E ATUALIZA A PLANILHA
		for res := range resultsChannel {
			if res.Err != nil {
				fmt.Printf("[%s] Error connecting to %s: %v\n", sheetName, res.IP, res.Err)
			} else {
				// Usa a res.Data
				fmt.Printf("[%s] %s - %s\n", sheetName, res.IP, res.Data)

				// Encontra a linha a qual esse IP pertence
				if rowIndex, ok := ipToRowIndex[res.IP]; ok {
					// Usa o ResultColumn
					cell := fmt.Sprintf("%s%d", ResultColumn, rowIndex)
					// Usa o res.Data
					if err := f.SetCellValue(sheetName, cell, res.Data); err != nil {
						log.Printf("[%s] Failed to write data '%s' to cell %s: %v", sheetName, res.Data, cell, err)
					}
				}
			}
		}
		fmt.Printf("--- Finished processing sheet: %s ---\n", sheetName)
	}

	// 5. SALVA O ARQUIVO (depois que todas as folhas são processadas)
	if err := f.Save(); err != nil {
		log.Fatalf("Failed to save Excel file: %v", err)
	}
	fmt.Printf("\nAll sheets processed. Result saved in \"%s\"\n", ExcelFile)
}

// getDeviceData conecta a um IP, roda o comando, e manda o resultado para um canal.
func getDeviceData(ip, username, password string, wg *sync.WaitGroup, ch chan<- Result, re *regexp.Regexp) {
	defer wg.Done() // Isso vai ser chamado quando a função sair, diminuindo o contador WaitGroup.

	// Configurar SSH client
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Equivale ao AutoAddPolicy
	}

	// Conecta ao servidor SSH
	client, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		ch <- Result{IP: ip, Err: err}
		return
	}
	defer client.Close()

	// Cria uma nova sessão
	session, err := client.NewSession()
	if err != nil {
		ch <- Result{IP: ip, Err: err}
		return
	}
	defer session.Close()

	// Rodar o comando e pegar o output
	output, err := session.CombinedOutput(Command)
	if err != nil {
		ch <- Result{IP: ip, Err: fmt.Errorf("failed to run command: %w", err)}
		return
	}

	// Parsear o output com o regex
	matches := re.FindStringSubmatch(string(output))

	if len(matches) > 1 {
		// Enviar o resultado para o campo 'Data'
		ch <- Result{IP: ip, Data: matches[1]}
	} else {
		ch <- Result{IP: ip, Err: errors.New(RegexError)}
	}
}
