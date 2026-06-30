package gui

import (
	"sort"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

// EnvironmentService wraps environment parsing for the Wails GUI.
type EnvironmentService struct {
	currentEnv string
}

// LoadEnvironments loads the environment file from the given directory.
// Returns the full EnvironmentFile with shared vars and per-env vars.
func (s *EnvironmentService) LoadEnvironments(dir string) (*model.EnvironmentFile, error) {
	return parser.LoadEnvironments(dir)
}

// ListEnvironmentNames returns a sorted list of environment names available in dir.
func (s *EnvironmentService) ListEnvironmentNames(dir string) ([]string, error) {
	envFile, err := parser.LoadEnvironments(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(envFile.Environments))
	for name := range envFile.Environments {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// ResolveVars returns all resolved variables for the given environment.
// Merges shared vars with env-specific vars.
func (s *EnvironmentService) ResolveVars(dir, envName string) (map[string]string, error) {
	envFile, err := parser.LoadEnvironments(dir)
	if err != nil {
		return nil, err
	}
	return parser.ResolveEnvironment(envFile, envName)
}

// GetCurrentEnv returns the currently selected environment name.
func (s *EnvironmentService) GetCurrentEnv() string {
	return s.currentEnv
}

// SetCurrentEnv sets the active environment.
func (s *EnvironmentService) SetCurrentEnv(name string) {
	s.currentEnv = name
}
