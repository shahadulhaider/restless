package writer

import (
	"errors"
	"fmt"
	"os"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

// InsertRequest appends a new request to the end of filePath.
// If the file does not exist it is created.
func InsertRequest(filePath string, req model.Request) error {
	existing, err := parser.ParseFile(filePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s: %w", filePath, err)
	}
	// Clear source metadata — this is a new logical request in the file.
	req.SourceFile = ""
	req.SourceLine = 0
	existing = append(existing, req)
	return writeRequests(filePath, existing)
}

// UpdateRequest replaces the request at oldReq.SourceLine with newReq in filePath.
// The match is done by SourceLine (the line number of the method token).
func UpdateRequest(filePath string, oldReq, newReq model.Request) error {
	existing, err := parser.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}
	found := false
	for i, r := range existing {
		if r.SourceLine == oldReq.SourceLine {
			newReq.SourceFile = ""
			newReq.SourceLine = 0
			existing[i] = newReq
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("request at line %d not found in %s", oldReq.SourceLine, filePath)
	}
	return writeRequests(filePath, existing)
}

// DeleteRequest removes the request at req.SourceLine from filePath.
// If the file becomes empty after deletion, an empty file is written.
func DeleteRequest(filePath string, req model.Request) error {
	existing, err := parser.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}
	filtered := existing[:0]
	found := false
	for _, r := range existing {
		if r.SourceLine == req.SourceLine {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if !found {
		return fmt.Errorf("request at line %d not found in %s", req.SourceLine, filePath)
	}
	return writeRequests(filePath, filtered)
}

// DuplicateRequest copies req to the end of dstFile.
// dstFile may be the same as srcFile (the request's SourceFile) or a different file.
func DuplicateRequest(req model.Request, dstFile string) error {
	// Strip source location — the duplicate is a new entry.
	req.SourceFile = ""
	req.SourceLine = 0
	return InsertRequest(dstFile, req)
}

// writeRequests serializes reqs and writes them to filePath.
// An empty slice writes an empty file.
func writeRequests(filePath string, reqs []model.Request) error {
	var content string
	if len(reqs) > 0 {
		content = SerializeRequests(reqs)
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}
