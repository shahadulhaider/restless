package writer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

// InsertRequest appends req to the end of filePath, creating the file if absent.
// Existing bytes are preserved verbatim (re-serializing the whole file would
// drop file-level @var lines and comments — defect D01).
func InsertRequest(filePath string, req model.Request) error {
	req.SourceFile = ""
	req.SourceLine = 0
	serialized := SerializeRequest(req)

	raw, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return atomicWriteFile(filePath, []byte(serialized+"\n"))
		}
		return fmt.Errorf("read %s: %w", filePath, err)
	}

	existing := strings.TrimRight(string(raw), " \t\r\n")
	if existing == "" {
		return atomicWriteFile(filePath, []byte(serialized+"\n"))
	}

	var sb strings.Builder
	sb.WriteString(existing)
	sb.WriteString("\n\n###\n\n")
	sb.WriteString(serialized)
	sb.WriteString("\n")
	return atomicWriteFile(filePath, []byte(sb.String()))
}

// UpdateRequest replaces the request at oldReq.SourceLine with newReq in filePath.
// The file preamble (file-level @var lines + header comments) is preserved (D01).
func UpdateRequest(filePath string, oldReq, newReq model.Request) error {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}
	existing, err := parser.ParseBytes(raw, filePath)
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
	return writeRequests(filePath, extractFilePreamble(raw), existing)
}

// DeleteRequest removes the request at req.SourceLine from filePath, preserving
// the file preamble (D01).
func DeleteRequest(filePath string, req model.Request) error {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}
	existing, err := parser.ParseBytes(raw, filePath)
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
	return writeRequests(filePath, extractFilePreamble(raw), filtered)
}

// DuplicateRequest copies req to the end of dstFile.
func DuplicateRequest(req model.Request, dstFile string) error {
	req.SourceFile = ""
	req.SourceLine = 0
	return InsertRequest(dstFile, req)
}

func isVarLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "@") && strings.Contains(trimmed, "=")
}

func isStandaloneComment(trimmed string) bool {
	return (strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//")) &&
		!strings.HasPrefix(trimmed, "# @") && !strings.HasPrefix(trimmed, "// @")
}

// extractFilePreamble recaptures file-scope content the parser discards so it
// survives an Update/Delete rewrite (D01): the leading block (verbatim) plus any
// file-level @var declared later between requests. fileScope tracks whether we
// sit after a ### and before the next request line, so request bodies are never
// mistaken for variable declarations.
func extractFilePreamble(raw []byte) string {
	normalized := strings.ReplaceAll(string(raw), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")

	var preamble []string
	i := 0
	for ; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" && !isVarLine(trimmed) && !isStandaloneComment(trimmed) {
			break
		}
		preamble = append(preamble, lines[i])
	}

	fileScope := false
	for ; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		switch {
		case strings.HasPrefix(trimmed, "###"):
			fileScope = true
		case trimmed == "":
		case fileScope && isVarLine(trimmed):
			preamble = append(preamble, lines[i])
		case fileScope && isStandaloneComment(trimmed):
		default:
			fileScope = false
		}
	}
	return strings.Join(preamble, "\n")
}

func writeRequests(filePath, preamble string, reqs []model.Request) error {
	var sb strings.Builder
	if p := strings.TrimRight(preamble, "\n"); p != "" {
		sb.WriteString(p)
		sb.WriteString("\n\n")
	}
	if len(reqs) > 0 {
		sb.WriteString(SerializeRequests(reqs))
		sb.WriteString("\n")
	}
	return atomicWriteFile(filePath, []byte(sb.String()))
}

// atomicWriteFile writes data to a temp file in the same directory, fsyncs, and
// renames it over filePath, so a crash mid-write cannot truncate the user's
// .http source of truth (D01).
func atomicWriteFile(filePath string, data []byte) error {
	dir := filepath.Dir(filePath)
	tmp, err := os.CreateTemp(dir, ".restless-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		return err
	}
	return os.Rename(tmpName, filePath)
}
