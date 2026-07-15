package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestLikeEscapingEndToEnd(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, 1, "50% concluído", "Percent", "")
	insertTestRecord(t, db, 2, "500 concluído", "Other", "")
	insertTestRecord(t, db, 3, "ana-c-silva", "Ana", "C.")
	insertTestRecord(t, db, 4, "diana-c-silva", "Diana", "C.")
	insertTestRecord(t, db, 5, "ana-ac-silva", "Ana", "AC.")
	insertTestRecord(t, db, 6, "ana-c-santos", "Ana", "C.")
	insertTestRecord(t, db, 7, "ana-c-lima", "Ana", "C.")
	setTestLastName(t, db, 3, "Silva")
	setTestLastName(t, db, 4, "Silva")
	setTestLastName(t, db, 5, "Silva")
	setTestLastName(t, db, 6, "Santos")
	setTestLastName(t, db, 7, "Lima")

	tests := []struct {
		name string
		run  func(context.Context, *sql.DB, string) ([]Record, error)
		in   string
		want string
		not  string
	}{
		{"percentual em identificador", byPartialIdentifier, "50%", "1: [50% concluído]", "2: [500 concluído]"},
		{"nome e middle exatos", byName, "Ana C.", "3: [ana-c-silva]", "4: [diana-c-silva]"},
		{"nome middle e sobrenome exatos", byName, "Ana C. Silva", "3: [ana-c-silva]", "5: [ana-ac-silva]"},
		{"prefixo do sobrenome", byName, "Ana C. L", "7: [ana-c-lima]", "4: [diana-c-silva]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := tt.run(context.Background(), db, tt.in)
			if err != nil {
				t.Fatal(err)
			}
			var gotBuf strings.Builder
			for _, r := range records {
				gotBuf.WriteString(r.String())
				gotBuf.WriteString("\n")
			}
			got := gotBuf.String()
			if !strings.Contains(got, tt.want) {
				t.Fatalf("resultado literal ausente para %q:\n%s", tt.in, got)
			}
			if strings.Contains(got, tt.not) {
				t.Fatalf("resultado obtido via wildcard para %q:\n%s", tt.in, got)
			}
		})
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("CREATE TABLE progresso (" +
		"num_combinacao INTEGER NOT NULL, combinacao TEXT NOT NULL, " +
		"nome TEXT, middle TEXT, sobrenome TEXT)"); err != nil {
		t.Fatal(err)
	}
	return db
}

func insertTestRecord(t *testing.T, db *sql.DB, id int, combination, name, middle string) {
	t.Helper()
	if _, err := db.Exec(
		"INSERT INTO progresso (num_combinacao, combinacao, nome, middle, sobrenome) VALUES (?, ?, ?, ?, '')",
		id, combination, name, middle,
	); err != nil {
		t.Fatal(err)
	}
}

func setTestLastName(t *testing.T, db *sql.DB, id int, lastName string) {
	t.Helper()
	if _, err := db.Exec("UPDATE progresso SET sobrenome = ? WHERE num_combinacao = ?", lastName, id); err != nil {
		t.Fatal(err)
	}
}

func TestByExactID(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, 1, "1", "Ana", "")

	tests := []struct {
		id   int64
		want string
	}{
		{1, "1: [1] -> Ana"},
		{999, ""},
	}
	for _, tt := range tests {
		records, err := byExactID(context.Background(), db, tt.id)
		if err != nil {
			t.Fatal(err)
		}
		var got string
		if len(records) > 0 {
			got = records[0].String()
		}
		if tt.want != "" && !strings.Contains(got, tt.want) {
			t.Fatalf("saída para id %d = %q; esperava conter %q", tt.id, got, tt.want)
		}
		if tt.want == "" && got != "" {
			t.Fatalf("saída para id %d = %q; esperava vazio", tt.id, got)
		}
	}
}

func TestByNameRejectsExcessSearchTerms(t *testing.T) {
	db := newTestDB(t)
	input := strings.Repeat("termo ", maxSearchTerms) + "extra"

	_, err := byName(context.Background(), db, input)
	if err == nil {
		t.Fatal("esperava erro de excesso de termos")
	}
	want := "Use no máximo 5 termos na busca."
	if err.Error() != want {
		t.Fatalf("erro = %q; esperava %q", err.Error(), want)
	}
}

func TestSearchRouting(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, 1, "0 1 2", "Ana", "C.")
	setTestLastName(t, db, 1, "Silva")

	// 1. Mixed query "4500í" should route to byName (indexed, fast) and return 0 results
	records, err := search(context.Background(), db, "4500í")
	if err != nil {
		t.Fatalf("search(4500í) failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 results, got %v", records)
	}

	// 2. Pure name "Ana" should route to byName and find the record
	records, err = search(context.Background(), db, "Ana")
	if err != nil {
		t.Fatalf("search(Ana) failed: %v", err)
	}
	if len(records) != 1 || records[0].FirstName != "Ana" {
		t.Errorf("expected Ana, got %v", records)
	}

	// 3. Combination snippet "0 1" should route to byPartialIdentifier and find the record
	records, err = search(context.Background(), db, "0 1")
	if err != nil {
		t.Fatalf("search(0 1) failed: %v", err)
	}
	if len(records) != 1 || records[0].Combination != "0 1 2" {
		t.Errorf("expected '0 1 2', got %v", records)
	}

	// 4. Wildcard queries like "_43" and "_43%" should route to byName and return 0 results instantly
	for _, q := range []string{"_43", "_43%", "1 2 999"} {
		records, err = search(context.Background(), db, q)
		if err != nil {
			t.Fatalf("search(%q) failed: %v", q, err)
		}
		if len(records) != 0 {
			t.Errorf("expected 0 results for %q, got %v", q, records)
		}
	}
}
