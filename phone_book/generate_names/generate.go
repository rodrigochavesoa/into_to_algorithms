package main

import (
	"database/sql"
	"flag"
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

const defaultDBFile = "phone_book/generate_names/db/bigon_bookV.db"

var db *sql.DB
var sinalInterrupcao chan os.Signal

func main() {
	dbPath := flag.String("db", defaultDBFile, "caminho para o banco SQLite")
	flag.Parse()

	n := 8 // Para 8, o total de combinações é 8! = 40320.

	var err error
	// Assegura que a pasta do DB existe
	err = os.MkdirAll(filepath.Dir(*dbPath), 0755)
	if err != nil {
		log.Fatalf("Erro criando pasta do DB: %v", err)
	}

	db, err = sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS progresso (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		num_combinacao BIGINT NOT NULL UNIQUE,
		combinacao TEXT NOT NULL,
		nome TEXT NOT NULL,
		middle TEXT NOT NULL,
		sobrenome TEXT NOT NULL
	);`)
	if err != nil {
		log.Fatalf("Erro ao iniciar o banco: %v", err)
	}
	if err := ensureSearchIndexes(db); err != nil {
		log.Fatalf("Erro ao criar índices de busca: %v", err)
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

	// listas simples de nomes, middle initials e sobrenomes (expandir conforme necessário)
	firstNames := []string{"Ana", "Bruno", "Carlos", "Diana", "Eduardo", "Fabiana", "Gustavo", "Helena", "Igor", "Juliana"}
	lastNames := []string{"Silva", "Santos", "Oliveira", "Souza", "Pereira", "Lima", "Gomes", "Ribeiro", "Ferreira", "Almeida"}
	middles := []string{"A.", "B.", "C.", "D.", "E.", "F.", "G.", "H.", "I.", "J."} // iniciais reduzidas

	// alvo desejado de nomes únicos (ex.: 44000)
	const desiredUniqueNames int64 = 44000
	maxCombinations := factorial(n)
	if desiredUniqueNames > maxCombinations {
		log.Printf("alvo de %d nomes é maior que as %d permutações possíveis para N=%d", desiredUniqueNames, maxCombinations, n)
	}

	// verificar capacidade F*L*M e expandir middles dinamicamente se necessário
	F := int64(len(firstNames))
	L := int64(len(lastNames))
	M := int64(len(middles))
	capacity := F * L * M
	if capacity < desiredUniqueNames {
		// calcular M necessário
		needM := (desiredUniqueNames + (F * L) - 1) / (F * L) // ceil(desired/(F*L))
		// expandir middles até needM (gera A., B., ..., Z., AA., AB., ...)
		middles = expandMiddles(middles, needM)
		M = int64(len(middles))
		capacity = F * L * M
		if capacity < desiredUniqueNames {
			log.Fatalf("Ainda incapaz de alcançar desired %d mesmo após expandir middles; capacidade=%d", desiredUniqueNames, capacity)
		}
		log.Printf("Expandido middles para %d entradas; nova capacidade=%d", len(middles), capacity)
	}

	if últimoNum > 0 {
		fmt.Printf("📦 Checkpoint encontrado! Retomando a partir da combinação nº %d\n", últimoNum)
		arr = stringToSlice(últimaString)
		temProxima := proximaPermutacao(arr)
		if !temProxima {
			fmt.Println("🎉 Todas as combinações possíveis já foram processadas e salvas!")
			return
		}
	} else {
		for i := range arr {
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

	// Gerar até atingir desiredUniqueNames (a partir do checkpoint)
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
		// interrompe quando já geramos o suficiente a partir do checkpoint
		if (totalCombinacoesEncontradas - últimoNum) > desiredUniqueNames {
			break
		}
		arrString := sliceToString(arr)

		// Gerar nome/middle/sobrenome determinístico a partir do número da combinação
		idx := totalCombinacoesEncontradas - 1
		F := int64(len(firstNames))
		L := int64(len(lastNames))
		M := int64(len(middles))

		iF := idx % F
		iL := (idx / F) % L
		iM := (idx / (F * L)) % M

		fn := firstNames[int(iF)]
		mi := middles[int(iM)]
		ln := lastNames[int(iL)]

		fmt.Printf("Combinação nº %d atingida: [%s] -> %s %s %s\n", totalCombinacoesEncontradas, arrString, fn, mi, ln)
		if _, err := db.Exec("INSERT INTO progresso (num_combinacao, combinacao, nome, middle, sobrenome) VALUES (?, ?, ?, ?, ?)", totalCombinacoesEncontradas, arrString, fn, mi, ln); err != nil {
			log.Fatalf("Erro ao salvar combinação nº %d: %v", totalCombinacoesEncontradas, err)
		}

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

func ensureSearchIndexes(db *sql.DB) error {
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_progresso_nome_nocase ON progresso(nome COLLATE NOCASE)",
		"CREATE INDEX IF NOT EXISTS idx_progresso_middle_nocase ON progresso(middle COLLATE NOCASE)",
		"CREATE INDEX IF NOT EXISTS idx_progresso_sobrenome_nocase ON progresso(sobrenome COLLATE NOCASE)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_progresso_nome_completo_nocase ON progresso(nome COLLATE NOCASE, middle COLLATE NOCASE, sobrenome COLLATE NOCASE)",
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func factorial(n int) int64 {
	result := int64(1)
	for i := 2; i <= n; i++ {
		result *= int64(i)
	}
	return result
}

// expandMiddles expande a slice inicial de middles para ter pelo menos targetM elementos.
// Gera rótulos como A., B., ..., Z., AA., AB., ... usando letras A-Z.
func expandMiddles(base []string, targetM int64) []string {
	letters := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	out := make([]string, 0, targetM)
	seen := make(map[string]struct{}, targetM)
	for _, v := range base {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
		if int64(len(out)) >= targetM {
			return out
		}
	}

	// gerar sequências de comprimento crescente
	length := 1
	for int64(len(out)) < targetM {
		// generate all combinations of given length
		indexes := make([]int, length)
		for {
			// build string
			var b strings.Builder
			for i := 0; i < length; i++ {
				b.WriteRune(letters[indexes[i]])
			}
			b.WriteString(".")
			candidate := b.String()
			if _, ok := seen[candidate]; !ok {
				seen[candidate] = struct{}{}
				out = append(out, candidate)
			}
			if int64(len(out)) >= targetM {
				return out
			}
			// increment indexes
			pos := length - 1
			for pos >= 0 {
				indexes[pos]++
				if indexes[pos] < len(letters) {
					break
				}
				indexes[pos] = 0
				pos--
			}
			if pos < 0 {
				break // exhausted this length
			}
		}
		length++
	}
	return out
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
