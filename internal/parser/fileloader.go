package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shahadulhaider/restless/internal/model"
)

func LoadFileBody(req *model.Request, collectionRoot string) (*model.Request, error) {
	if req.BodyFile == "" {
		return req, nil
	}

	sourceDir := filepath.Dir(req.SourceFile)
	absPath, err := filepath.Abs(filepath.Join(sourceDir, req.BodyFile))
	if err != nil {
		return nil, fmt.Errorf("resolving path %q: %w", req.BodyFile, err)
	}

	absRoot, err := filepath.Abs(collectionRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving collection root: %w", err)
	}

	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || len(rel) >= 2 && rel[:2] == ".." {
		return nil, fmt.Errorf("path %q traverses outside collection root", req.BodyFile)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("%s:%d: cannot read body file %q: %w", req.SourceFile, req.SourceLine, req.BodyFile, err)
	}

	resolved := *req
	resolved.Body = string(data)
	resolved.BodyFile = ""
	return &resolved, nil
}
