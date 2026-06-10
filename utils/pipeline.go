// utils/pipeline.go
package utils

import "to-do/models"

// C.62 — Concurrency Pattern: Pipeline
// C.63 — Fan-in Fan-out

// TodoGenerator - Stage 1: Generator — kirim todos ke channel
func TodoGenerator(todos []models.Todo) <-chan models.Todo {
	out := make(chan models.Todo)
	go func() {
		for _, t := range todos {
			out <- t
		}
		close(out) // A.34 — Channel Range & Close
	}()
	return out
}

// FilterPending - Stage 2: Filter — hanya loloskan todo yang belum selesai
func FilterPending(in <-chan models.Todo) <-chan models.Todo {
	out := make(chan models.Todo)
	go func() {
		for todo := range in { // A.34 — range atas channel
			if todo.Status != models.StatusDone {
				out <- todo
			}
		}
		close(out)
	}()
	return out
}

// EnrichWithOverdue - Stage 3: Enricher — tambah info overdue
func EnrichWithOverdue(in <-chan models.Todo) <-chan models.Todo {
	out := make(chan models.Todo)
	go func() {
		for todo := range in {
			// Data enrichment (A.40 — Time)
			_ = todo.IsOverdue()
			out <- todo
		}
		close(out)
	}()
	return out
}
