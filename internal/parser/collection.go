package parser

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shahadulhaider/restless/internal/model"
)

func LoadCollection(rootDir string) (*model.Collection, error) {
	c := &model.Collection{RootDir: rootDir}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".http") {
			return nil
		}
		reqs, parseErr := ParseFile(path)
		if parseErr != nil {
			return nil
		}
		c.Files = append(c.Files, model.HTTPFile{Path: path, Requests: reqs})
		return nil
	})

	sort.Slice(c.Files, func(i, j int) bool {
		return c.Files[i].Path < c.Files[j].Path
	})

	return c, err
}
