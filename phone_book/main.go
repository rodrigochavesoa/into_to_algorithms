package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultDBFile  = "phone_book/generate_names/db/bigon_bookX.db"
	maxInputBytes  = 4 << 10
	maxSearchTerms = 5
)

// Record representa uma linha da tabela progresso.
type Record struct {
	CombinationNum int64
	Combination    string
	FirstName      string
	MiddleName     sql.NullString
	LastName       string
}

func (r Record) String() string {
	if r.MiddleName.Valid && r.MiddleName.String != "" {
		return fmt.Sprintf("%d: [%s] -> %s %s %s", r.CombinationNum, r.Combination, r.FirstName, r.MiddleName.String, r.LastName)
	}
	return fmt.Sprintf("%d: [%s] -> %s %s", r.CombinationNum, r.Combination, r.FirstName, r.LastName)
}

func main() {
	dbPath := flag.String("db", defaultDBFile, "caminho para o banco SQLite")
	flag.Parse()

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("erro ao abrir banco de dados %q: %v", *dbPath, err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("banco de dados %q inacessível: %v", *dbPath, err)
	}

	fmt.Println("Busca no banco de combinações")
	fmt.Println("Digite um id, nome e/ou sobrenome:")
	fmt.Print("> ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024), maxInputBytes)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Printf("erro ao ler entrada: %v", err)
			return
		}
		fmt.Println("entrada inválida ou cancelada")
		return
	}

	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		fmt.Println("entrada vazia")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := search(ctx, db, os.Stdout, input); err != nil {
		log.Fatalf("erro durante a busca: %v", err)
	}
}

// search decide a estratégia de busca a partir do formato da entrada.
func search(ctx context.Context, db *sql.DB, w io.Writer, input string) error {
	switch {
	case isAllDigits(input):
		num, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido: %w", err)
		}
		return byExactID(ctx, db, w, num)
	case containsDigit(input):
		return byPartialIdentifier(ctx, db, w, input)
	default:
		return byName(ctx, db, w, input)
	}
}

func byExactID(ctx context.Context, db *sql.DB, w io.Writer, num int64) error {
	const q = `SELECT num_combinacao, combinacao, nome, middle, sobrenome FROM progresso WHERE num_combinacao = ?`
	var r Record
	err := db.QueryRowContext(ctx, q, num).Scan(&r.CombinationNum, &r.Combination, &r.FirstName, &r.MiddleName, &r.LastName)
	if err == sql.ErrNoRows {
		fmt.Fprintln(w, "nenhum resultado para esse número exato")
		return nil
	}
	if err != nil {
		return fmt.Errorf("busca por id: %w", err)
	}
	fmt.Fprintln(w, r.String())
	return nil
}

// byName interpreta a entrada na ordem convencional: nome, middle e sobrenome.
// A comparação é exata e case-insensitive para não confundir "Ana" com "Diana".
func byName(ctx context.Context, db *sql.DB, w io.Writer, input string) error {
	terms := strings.Fields(input)
	if len(terms) > maxSearchTerms {
		_, err := fmt.Fprintf(w, "Use no máximo %d termos na busca.\n", maxSearchTerms)
		return err
	}

	const selectPrefix = `SELECT num_combinacao, combinacao, nome, middle, sobrenome
		FROM progresso
		WHERE `
	const limit = ` LIMIT 200`

	switch len(terms) {
	case 1:
		query := selectPrefix + `
			nome = ? COLLATE NOCASE OR middle = ? COLLATE NOCASE OR sobrenome = ? COLLATE NOCASE` + limit
		return runQuery(ctx, db, w, query, terms[0], terms[0], terms[0])
	case 2:
		column := "sobrenome"
		if strings.HasSuffix(terms[1], ".") {
			column = "middle"
		}
		if column == "middle" {
			query := selectPrefix + "nome = ? COLLATE NOCASE AND middle = ? COLLATE NOCASE" + limit
			return runQuery(ctx, db, w, query, terms[0], terms[1])
		}
		query := selectPrefix + "nome = ? COLLATE NOCASE AND sobrenome LIKE ? ESCAPE '\\'" + limit
		return runQuery(ctx, db, w, query, terms[0], likePrefix(terms[1]))
	default:
		lastName := strings.Join(terms[2:], " ")
		query := selectPrefix + `
			nome = ? COLLATE NOCASE
			AND middle = ? COLLATE NOCASE
			AND sobrenome LIKE ? ESCAPE '\'` + limit
		return runQuery(ctx, db, w, query, terms[0], terms[1], likePrefix(lastName))
	}
}

func byPartialIdentifier(ctx context.Context, db *sql.DB, w io.Writer, input string) error {
	const query = `SELECT num_combinacao, combinacao, nome, middle, sobrenome
		FROM progresso
		WHERE CAST(num_combinacao AS TEXT) LIKE ? ESCAPE '\' OR combinacao LIKE ? ESCAPE '\'
		LIMIT 200`
	pattern := likeContains(input)
	return runQuery(ctx, db, w, query, pattern, pattern)
}

// likeContains preserva a intenção de busca literal: %, _ e \\ digitados pelo
// usuário não passam a significar curingas do SQL.
func likeContains(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return "%" + s + "%"
}

func likePrefix(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s + "%"
}

func runQuery(ctx context.Context, db *sql.DB, w io.Writer, query string, args ...any) error {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.CombinationNum, &r.Combination, &r.FirstName, &r.MiddleName, &r.LastName); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		fmt.Fprintln(w, r.String())
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iteração do cursor: %w", err)
	}
	if count == 0 {
		fmt.Fprintln(w, "nenhum resultado encontrado")
	}
	return nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
