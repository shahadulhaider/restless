package gui

import (
	"fmt"

	"github.com/shahadulhaider/restless/internal/assert"
	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/history"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/shahadulhaider/restless/internal/script"
)

// RequestService handles HTTP request execution for the GUI.
type RequestService struct {
	chainCtx *parser.ChainContext
	cookies  *engine.CookieManager
	rootDir  string
}

// NewRequestService creates a RequestService with initialised chain and cookie state.
func NewRequestService() *RequestService {
	return &RequestService{
		chainCtx: parser.NewChainContext(),
		cookies:  engine.NewCookieManager(),
	}
}

// SetRootDir sets the collection root directory.
func (s *RequestService) SetRootDir(dir string) {
	s.rootDir = dir
}

// Execute sends an HTTP request with the full pipeline:
//  1. Resolve environment variables
//  2. Merge env vars + file-level inline variables
//  3. Resolve variables in request (URL, headers, body)
//  4. Load file body if @file reference exists
//  5. Run pre-request script (if any)
//  6. Execute HTTP request via engine.ExecuteWithJar
//  7. Run post-response script (if any)
//  8. Evaluate assertions
//  9. Store in chain context for chaining
//  10. Save to history
func (s *RequestService) Execute(req model.Request, envName string) (*model.Response, error) {
	envVars, err := s.resolveEnvVars(envName)
	if err != nil {
		return nil, fmt.Errorf("resolving environment: %w", err)
	}

	mergedVars := s.mergeVars(envVars, req.SourceFile)

	resolved, _ := parser.ResolveRequest(&req, mergedVars, s.chainCtx)

	loaded, err := parser.LoadFileBody(resolved, s.rootDir)
	if err != nil {
		loaded = resolved
	}

	if loaded.PreRequestScript != "" {
		scriptCtx := &script.ScriptContext{
			Request: loaded,
			EnvVars: mergedVars,
		}
		if scriptErr := script.RunPreRequest(loaded.PreRequestScript, scriptCtx); scriptErr != nil {
			return nil, fmt.Errorf("pre-request script: %w", scriptErr)
		}
	}

	jar := s.cookies.JarForEnv(envName)
	resp, err := engine.ExecuteWithJar(loaded, jar)
	if err != nil {
		return nil, err
	}

	if loaded.PostResponseScript != "" {
		scriptCtx := &script.ScriptContext{
			Request:  loaded,
			Response: resp,
			EnvVars:  mergedVars,
		}
		if scriptErr := script.RunPostResponse(loaded.PostResponseScript, scriptCtx); scriptErr != nil {
			resp.ScriptError = scriptErr.Error()
		}
	}

	if len(loaded.Assertions) > 0 {
		resp.AssertionResults = assert.EvaluateAll(loaded, resp)
	}

	if req.Name != "" {
		s.chainCtx.StoreResponse(req.Name, resp)
	}

	if s.rootDir != "" {
		_ = history.Save(s.rootDir, &req, resp, envName)
	}

	return resp, nil
}

// ResolvePreview resolves all variables in the request WITHOUT sending it.
// Shows what the final URL, headers and body would look like.
func (s *RequestService) ResolvePreview(req model.Request, envName string) (*model.Request, error) {
	envVars, err := s.resolveEnvVars(envName)
	if err != nil {
		return nil, fmt.Errorf("resolving environment: %w", err)
	}

	mergedVars := s.mergeVars(envVars, req.SourceFile)

	resolved, err := parser.ResolveRequest(&req, mergedVars, s.chainCtx)
	if err != nil {
		return nil, fmt.Errorf("resolving request: %w", err)
	}

	loaded, err := parser.LoadFileBody(resolved, s.rootDir)
	if err != nil {
		return resolved, nil
	}

	return loaded, nil
}

// GetChainVariables returns the names of requests whose responses are stored
// in the chain context (available for {{name.response.body.field}} syntax).
func (s *RequestService) GetChainVariables() []string {
	names := make([]string, 0, len(s.chainCtx.Responses))
	for n := range s.chainCtx.Responses {
		names = append(names, n)
	}
	return names
}

func (s *RequestService) resolveEnvVars(envName string) (map[string]string, error) {
	if s.rootDir == "" {
		return make(map[string]string), nil
	}
	envFile, err := parser.LoadEnvironments(s.rootDir)
	if err != nil {
		return make(map[string]string), nil
	}
	return parser.ResolveEnvironment(envFile, envName)
}

func (s *RequestService) mergeVars(envVars map[string]string, sourceFile string) map[string]string {
	merged := make(map[string]string, len(envVars))
	for k, v := range envVars {
		merged[k] = v
	}
	if sourceFile != "" {
		if fileVars, err := parser.ExtractFileVariablesFromFile(sourceFile); err == nil {
			for k, v := range fileVars {
				if _, exists := merged[k]; !exists {
					merged[k] = v
				}
			}
		}
	}
	return merged
}
