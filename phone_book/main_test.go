package main

import (
	"bytes"
	"context"
	"database/sql"
	"io"
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
		run  func(context.Context, *sql.DB, io.Writer, string) error
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
			var output bytes.Buffer
			if err := tt.run(context.Background(), db, &output, tt.in); err != nil {
				t.Fatal(err)
			}
			got := output.String()
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

func TestByExactIDWritesToProvidedWriter(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, 1, "1", "Ana", "")

	tests := []struct {
		id   int64
		want string
	}{
		{1, "1: [1] -> Ana"},
		{999, "nenhum resultado para esse número exato"},
	}
	for _, tt := range tests {
		var output bytes.Buffer
		if err := byExactID(context.Background(), db, &output, tt.id); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.String(), tt.want) {
			t.Fatalf("saída para id %d = %q; esperava %q", tt.id, output.String(), tt.want)
		}
	}
}

func TestByNameRejectsExcessSearchTerms(t *testing.T) {
	db := newTestDB(t)
	input := strings.Repeat("termo ", maxSearchTerms) + "extra"
	var output bytes.Buffer

	if err := byName(context.Background(), db, &output, input); err != nil {
		t.Fatal(err)
	}
	want := "Use no máximo 5 termos na busca.\n"
	if output.String() != want {
		t.Fatalf("saída = %q; esperava %q", output.String(), want)
	}
}
