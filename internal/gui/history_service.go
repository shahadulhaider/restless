package gui

import (
	"github.com/shahadulhaider/restless/internal/history"
	"github.com/shahadulhaider/restless/internal/model"
)

// HistoryService exposes request history to the Wails GUI.
type HistoryService struct{}

// List returns history entries for a given request.
func (s *HistoryService) List(rootDir string, req *model.Request) ([]history.HistoryEntry, error) {
	return history.List(rootDir, req)
}

// Diff returns a text diff between two history entries.
func (s *HistoryService) Diff(a, b *history.HistoryEntry) string {
	return history.Diff(a, b)
}
