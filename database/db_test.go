package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigrateAddsSortOrderBeforeCreatingIndex(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.Exec(`
		CREATE TABLE todos (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			priority TEXT DEFAULT 'medium',
			status TEXT DEFAULT 'pending',
			due_date DATETIME,
			tags TEXT DEFAULT '[]',
			sub_tasks TEXT DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	(&DB{Conn: conn}).migrate()

	var sortOrderColumnCount int
	err = conn.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name = 'sort_order'
	`).Scan(&sortOrderColumnCount)
	if err != nil {
		t.Fatal(err)
	}
	if sortOrderColumnCount != 1 {
		t.Fatalf("expected sort_order column, got count %d", sortOrderColumnCount)
	}

	var sortOrderIndexCount int
	err = conn.QueryRow(`
		SELECT COUNT(*) FROM pragma_index_list('todos') WHERE name = 'idx_todos_sort_order'
	`).Scan(&sortOrderIndexCount)
	if err != nil {
		t.Fatal(err)
	}
	if sortOrderIndexCount != 1 {
		t.Fatalf("expected sort_order index, got count %d", sortOrderIndexCount)
	}
}
