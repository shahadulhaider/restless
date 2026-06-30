package gui

import (
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/shahadulhaider/restless/internal/writer"
)

// CollectionService wraps parser and writer packages for collection management
// in the Wails GUI.
type CollectionService struct{}

// LoadCollection loads all .http files from the root directory.
func (s *CollectionService) LoadCollection(dir string) (*model.Collection, error) {
	return parser.LoadCollection(dir)
}

// GetRequests parses a single .http file and returns its requests.
func (s *CollectionService) GetRequests(filePath string) ([]model.Request, error) {
	return parser.ParseFile(filePath)
}

// CreateRequest inserts a new request into the specified .http file.
func (s *CollectionService) CreateRequest(filePath string, req model.Request) error {
	return writer.InsertRequest(filePath, req)
}

// UpdateRequest updates an existing request in the file.
func (s *CollectionService) UpdateRequest(filePath string, old, updated model.Request) error {
	return writer.UpdateRequest(filePath, old, updated)
}

// DeleteRequest removes a request from the file.
func (s *CollectionService) DeleteRequest(filePath string, req model.Request) error {
	return writer.DeleteRequest(filePath, req)
}

// DuplicateRequest duplicates a request to the specified destination file.
func (s *CollectionService) DuplicateRequest(req model.Request, dstFile string) error {
	return writer.DuplicateRequest(req, dstFile)
}

// CreateFile creates a new .http file at the specified relative path.
func (s *CollectionService) CreateFile(root, relPath string) error {
	return writer.CreateHTTPFile(root, relPath)
}

// CreateDirectory creates a new directory under the root.
func (s *CollectionService) CreateDirectory(root, name string) error {
	return writer.CreateDirectory(root, name)
}

// RenameEntry renames a file or directory.
func (s *CollectionService) RenameEntry(root, oldRelPath, newRelPath string) error {
	return writer.RenameEntry(root, oldRelPath, newRelPath)
}

// DeleteEntry deletes a file or directory.
func (s *CollectionService) DeleteEntry(root, relPath string) error {
	return writer.DeleteEntry(root, relPath)
}
