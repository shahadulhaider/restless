package gui

import "github.com/shahadulhaider/restless/internal/importer"

// ImporterService wraps the importer package for Wails GUI consumption.
type ImporterService struct{}

// ImportPostman imports a Postman collection file into .http files.
func (s *ImporterService) ImportPostman(file, outputDir string) error {
	return importer.ImportPostman(file, importer.ImportOptions{OutputDir: outputDir})
}

// ImportInsomnia imports an Insomnia collection export.
func (s *ImporterService) ImportInsomnia(file, outputDir string) error {
	return importer.ImportInsomnia(file, importer.ImportOptions{OutputDir: outputDir})
}

// ImportBruno imports a Bruno collection directory.
func (s *ImporterService) ImportBruno(dir, outputDir string) error {
	return importer.ImportBruno(dir, importer.ImportOptions{OutputDir: outputDir})
}

// ImportCurl imports a curl command string.
func (s *ImporterService) ImportCurl(command, outputDir string) error {
	return importer.ImportCurl(command, importer.ImportOptions{OutputDir: outputDir})
}

// ImportOpenAPI imports an OpenAPI/Swagger spec.
func (s *ImporterService) ImportOpenAPI(spec, outputDir string) error {
	return importer.ImportOpenAPI(spec, importer.ImportOptions{OutputDir: outputDir})
}
