-- Execute uma vez no banco de produção antes de depender da busca por nome.
-- Os índices usam NOCASE porque as consultas de byName também o usam.
CREATE INDEX IF NOT EXISTS idx_progresso_nome_nocase
    ON progresso(nome COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_progresso_middle_nocase
    ON progresso(middle COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_progresso_sobrenome_nocase
    ON progresso(sobrenome COLLATE NOCASE);
CREATE UNIQUE INDEX IF NOT EXISTS idx_progresso_nome_completo_nocase
    ON progresso(nome COLLATE NOCASE, middle COLLATE NOCASE, sobrenome COLLATE NOCASE);
