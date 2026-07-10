package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

const DbFile = "db/bigon_V000.db"

var db *sql.DB
var sinalInterrupcao chan os.Signal

func main() {
	n := 4 // Para 8, o total de combinações é (8! = 40320) 

	var err error
	// Assegura que a pasta do DB existe
	err = os.MkdirAll(filepath.Dir(DbFile), 0755)
	if err != nil {
		log.Fatalf("Erro criando pasta do DB: %v", err)
	}

	db, err = sql.Open("sqlite", DbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS progresso (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        num_combinacao BIGINT NOT NULL UNIQUE,
        combinacao TEXT NOT NULL
    );`)
	if err != nil {
		log.Fatalf("Erro ao iniciar o banco: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS execucoes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start_time TEXT NOT NULL,
		end_time TEXT,
		duration_minutes REAL,
		total_combinacoes BIGINT,
		status TEXT NOT NULL
	);`)
	if err != nil {
		log.Fatalf("Erro ao criar tabela de execucoes: %v", err)
	}

	var últimoNum int64 = 0
	var últimaString string = ""
	_ = db.QueryRow("SELECT num_combinacao, combinacao FROM progresso ORDER BY num_combinacao DESC LIMIT 1").Scan(&últimoNum, &últimaString)

	arr := make([]int, n)
	totalCombinacoesEncontradas := últimoNum

	if últimoNum > 0 {
		fmt.Printf("📦 Checkpoint encontrado! Retomando a partir da combinação nº %d\n", últimoNum)
		arr = stringToSlice(últimaString)
		temProxima := proximaPermutacao(arr)
		if !temProxima {
			fmt.Println("🎉 Todas as combinações possíveis já foram processadas e salvas!")
			return
		}
	} else {
		for i := range n { 
			arr[i] = i
		}
	}

	sinalInterrupcao = make(chan os.Signal, 1)
	signal.Notify(sinalInterrupcao, os.Interrupt, syscall.SIGTERM)

	var startTime time.Time
	startTime = time.Now()

	var runID int64 = 0

	fmt.Printf("🚀 Iniciando algoritmo resiliente para N=%d...\n", n)
	fmt.Println("------------------------------------------------------------------")

	// Registra execução no banco
	res, err := db.Exec("INSERT INTO execucoes (start_time, status) VALUES (?, ?)", startTime.Format(time.RFC3339), "running")
	if err != nil {
		log.Printf("Aviso: não foi possível registrar execução no DB: %v", err)
	} else {
		runID, _ = res.LastInsertId()
	}

	for {
		select {
		case <-sinalInterrupcao:
			endTime := time.Now()
			dur := endTime.Sub(startTime)
			fmt.Printf("\n🛑 Interrupção detectada! Último checkpoint garantido no banco: nº %d\n", totalCombinacoesEncontradas)
			fmt.Printf("Hora de início: %s\n", startTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("Hora de término: %s\n", endTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("Duração: %.2f minutos\n", dur.Minutes())
			if runID != 0 {
				_, _ = db.Exec("UPDATE execucoes SET end_time=?, duration_minutes=?, total_combinacoes=?, status=? WHERE id=?", endTime.Format(time.RFC3339), dur.Minutes(), totalCombinacoesEncontradas, "interrupted", runID)
			}
			os.Exit(0)
		default:
		}

		totalCombinacoesEncontradas++
		arrString := sliceToString(arr)

		fmt.Printf("Combinação nº %d atingida: [%s]\n", totalCombinacoesEncontradas, arrString)
		_, _ = db.Exec("INSERT OR IGNORE INTO progresso (num_combinacao, combinacao) VALUES (?, ?)", totalCombinacoesEncontradas, arrString)

		if !proximaPermutacao(arr) {
			break
		}
	}

	fmt.Println("------------------------------------------------------------------")
	fmt.Printf("🎉 Concluído com sucesso! Total de combinações salvas no histórico: %d\n", totalCombinacoesEncontradas)
	endTime := time.Now()
	dur := endTime.Sub(startTime)
	fmt.Printf("Hora de início: %s\n", startTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Hora de término: %s\n", endTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duração: %.2f minutos\n", dur.Minutes())
	if runID != 0 {
		_, _ = db.Exec("UPDATE execucoes SET end_time=?, duration_minutes=?, total_combinacoes=?, status=? WHERE id=?", endTime.Format(time.RFC3339), dur.Minutes(), totalCombinacoesEncontradas, "completed", runID)
	}
}

// Algoritmo de Narayana Pandita A decisão foi abandonar a recursão 
// (onde a função chama a si mesma) e adotar uma abordagem iterativa (baseada em um laço de repetição contínuo).
func proximaPermutacao(arr []int) bool {
	i := len(arr) - 2
	for i >= 0 && arr[i] >= arr[i+1] {
		i--
	}
	if i < 0 {
		return false // Chegou na última combinação possível.
	}

	j := len(arr) - 1
	for arr[j] <= arr[i] {
		j--
	}

	// Troca os elementos i e j
	arr[i], arr[j] = arr[j], arr[i]

	// Inverte o restante da sequência após a posição i
	left, right := i+1, len(arr)-1
	for left < right {
		arr[left], arr[right] = arr[right], arr[left]
		left++
		right--
	}
	return true
}

func sliceToString(arr []int) string {
	strVals := make([]string, len(arr))
	for i, v := range arr {
		strVals[i] = strconv.Itoa(v)
	}
	return strings.Join(strVals, " ")
}

func stringToSlice(str string) []int {
	fields := strings.Fields(str)
	arr := make([]int, len(fields))
	for i, f := range fields {
		arr[i], _ = strconv.Atoi(f)
	}
	return arr
}