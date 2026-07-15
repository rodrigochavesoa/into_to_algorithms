package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultDBFile  = "phone_book/generate_names/db/bigon_bookX.db"
	maxInputBytes  = 4 << 10
	maxQueryBytes  = 200
	maxSearchTerms = 5
)

// Record representa um registro da tabela progresso.
type Record struct {
	CombinationNum int64          `json:"combination_num"`
	Combination    string         `json:"combination"`
	FirstName      string         `json:"first_name"`
	MiddleName     sql.NullString `json:"-"`
	LastName       string         `json:"last_name"`
}

// MarshalJSON customiza a serialização do Record tratando MiddleName como string comum.
func (r Record) MarshalJSON() ([]byte, error) {
	type Alias Record
	return json.Marshal(&struct {
		Alias
		MiddleName string `json:"middle_name"`
	}{
		Alias:      (Alias)(r),
		MiddleName: r.MiddleName.String,
	})
}

func (r Record) String() string {
	if r.MiddleName.Valid && r.MiddleName.String != "" {
		return fmt.Sprintf("%d: [%s] -> %s %s %s", r.CombinationNum, r.Combination, r.FirstName, r.MiddleName.String, r.LastName)
	}
	return fmt.Sprintf("%d: [%s] -> %s %s", r.CombinationNum, r.Combination, r.FirstName, r.LastName)
}

func main() {
	dbPath := flag.String("db", defaultDBFile, "caminho para o banco SQLite")
	server := flag.Bool("server", false, "iniciar em modo servidor web")
	port := flag.String("port", "8080", "porta do servidor web")
	flag.Parse()

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("erro ao abrir banco de dados %q: %v", *dbPath, err)
	}
	defer db.Close()

	// Configurações do pool de conexões para o SQLite.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("banco de dados %q inacessível: %v", *dbPath, err)
	}

	if *server {
		startServer(db, *port)
		return
	}

	// Execução em modo linha de comando.
	fmt.Println("Busca no banco de combinações (Modo CLI)")
	fmt.Println("Digite um id, nome e/ou sobrenome:")
	fmt.Print("> ")

	var input string
	var buf [maxInputBytes]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil && err != io.EOF {
		log.Fatalf("erro ao ler entrada: %v", err)
	}
	input = strings.TrimSpace(string(buf[:n]))
	if input == "" {
		fmt.Println("entrada vazia")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	records, err := search(ctx, db, input)
	if err != nil {
		log.Fatalf("erro durante a busca: %v", err)
	}

	if len(records) == 0 {
		fmt.Println("nenhum resultado encontrado")
		return
	}

	for _, r := range records {
		fmt.Println(r.String())
	}
}

// search seleciona a estratégia de busca de acordo com o formato da entrada.
func search(ctx context.Context, db *sql.DB, input string) ([]Record, error) {
	if len(input) > maxInputBytes {
		return nil, fmt.Errorf("entrada excede o limite de tamanho permitido")
	}

	switch {
	case isAllDigits(input):
		num, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("id inválido: %w", err)
		}
		return byExactID(ctx, db, num)
	case isPartialIdentifier(input):
		return byPartialIdentifier(ctx, db, input)
	default:
		return byName(ctx, db, input)
	}
}

func byExactID(ctx context.Context, db *sql.DB, num int64) ([]Record, error) {
	const q = `SELECT num_combinacao, combinacao, nome, middle, sobrenome FROM progresso WHERE num_combinacao = ?`
	var r Record
	err := db.QueryRowContext(ctx, q, num).Scan(&r.CombinationNum, &r.Combination, &r.FirstName, &r.MiddleName, &r.LastName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("busca por id: %w", err)
	}
	return []Record{r}, nil
}

// byName busca registros associando os termos da entrada a nome, inicial/nome do meio e sobrenome.
func byName(ctx context.Context, db *sql.DB, input string) ([]Record, error) {
	terms := strings.Fields(input)
	if len(terms) > maxSearchTerms {
		return nil, fmt.Errorf("Use no máximo %d termos na busca.", maxSearchTerms)
	}

	const selectPrefix = `SELECT num_combinacao, combinacao, nome, middle, sobrenome
		FROM progresso
		WHERE `
	const limit = ` LIMIT 200`

	switch len(terms) {
	case 1:
		query := selectPrefix + `
			nome = ? COLLATE NOCASE OR middle = ? COLLATE NOCASE OR sobrenome = ? COLLATE NOCASE` + limit
		return runQuery(ctx, db, query, terms[0], terms[0], terms[0])
	case 2:
		column := "sobrenome"
		if strings.HasSuffix(terms[1], ".") {
			column = "middle"
		}
		if column == "middle" {
			query := selectPrefix + "nome = ? COLLATE NOCASE AND middle = ? COLLATE NOCASE" + limit
			return runQuery(ctx, db, query, terms[0], terms[1])
		}
		query := selectPrefix + "nome = ? COLLATE NOCASE AND sobrenome LIKE ? ESCAPE '\\'" + limit
		return runQuery(ctx, db, query, terms[0], likePrefix(terms[1]))
	default:
		lastName := strings.Join(terms[2:], " ")
		query := selectPrefix + `
			nome = ? COLLATE NOCASE
			AND middle = ? COLLATE NOCASE
			AND sobrenome LIKE ? ESCAPE '\'` + limit
		return runQuery(ctx, db, query, terms[0], terms[1], likePrefix(lastName))
	}
}

func byPartialIdentifier(ctx context.Context, db *sql.DB, input string) ([]Record, error) {
	const query = `SELECT num_combinacao, combinacao, nome, middle, sobrenome
		FROM progresso
		WHERE CAST(num_combinacao AS TEXT) LIKE ? ESCAPE '\' OR combinacao LIKE ? ESCAPE '\'
		LIMIT 200`
	pattern := likeContains(input)
	return runQuery(ctx, db, query, pattern, pattern)
}

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

func runQuery(ctx context.Context, db *sql.DB, query string, args ...any) ([]Record, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.CombinationNum, &r.Combination, &r.FirstName, &r.MiddleName, &r.LastName); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iteração do cursor: %w", err)
	}
	return records, nil
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

func isPartialIdentifier(s string) bool {
	hasDigit := false
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if len(token) != 1 {
			return false
		}
		r := token[0]
		if r < '0' || r > '9' {
			return false
		}
		hasDigit = true
	}
	return hasDigit && strings.Contains(s, " ")
}

// Servidor Web e Segurança

func startServer(db *sql.DB, port string) {
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		log.Fatalf("porta inválida %q: use um valor entre 1 e 65535", port)
	}

	mux := http.NewServeMux()

	limiter := newIPRateLimiter()
	limiter.startCleanup(1*time.Minute, 10*time.Second)

	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Método não permitido"})
			return
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		if !limiter.allow(ip, 30, 10*time.Second) {
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "Muitas requisições. Por favor, aguarde um momento."})
			return
		}

		q := r.URL.Query().Get("q")
		q = strings.TrimSpace(q)

		if q == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Termo de busca vazio"})
			return
		}

		if len(q) > maxQueryBytes {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Termo de busca muito longo"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		records, err := search(ctx, db, q)
		if err != nil {
			log.Printf("erro na busca: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Erro interno ao executar a busca"})
			return
		}

		if records == nil {
			records = []Record{}
		}

		json.NewEncoder(w).Encode(records)
	})

	// Serve arquivos estáticos da interface web.
	webDir := "web"
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		webDir = filepath.Clean("phone_book/web")
	}
	fs := http.FileServer(http.Dir(webDir))
	mux.Handle("/", fs)

	handler := securityHeaders(mux)
	addr := ":" + port
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	fmt.Printf("Servidor iniciado em http://%s\n", validHostPort(port))
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("erro ao iniciar o servidor: %v", err)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}

func validHostPort(port string) string {
	return net.JoinHostPort("localhost", port)
}

// ipRateLimiter gerencia a frequência de requisições por IP na memória.
type ipRateLimiter struct {
	ips map[string][]time.Time
	mu  sync.Mutex
}

func newIPRateLimiter() *ipRateLimiter {
	return &ipRateLimiter{ips: make(map[string][]time.Time)}
}

func (l *ipRateLimiter) allow(ip string, limit int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	times := l.ips[ip]

	var validTimes []time.Time
	for _, t := range times {
		if now.Sub(t) < window {
			validTimes = append(validTimes, t)
		}
	}

	if len(validTimes) >= limit {
		l.ips[ip] = validTimes
		return false
	}

	validTimes = append(validTimes, now)
	l.ips[ip] = validTimes
	return true
}

// startCleanup remove IPs inativos periodicamente para liberar memória.
func (l *ipRateLimiter) startCleanup(interval time.Duration, maxAge time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			l.mu.Lock()
			now := time.Now()
			for ip, times := range l.ips {
				var validTimes []time.Time
				for _, t := range times {
					if now.Sub(t) < maxAge {
						validTimes = append(validTimes, t)
					}
				}
				if len(validTimes) == 0 {
					delete(l.ips, ip)
				} else {
					l.ips[ip] = validTimes
				}
			}
			l.mu.Unlock()
		}
	}()
}
