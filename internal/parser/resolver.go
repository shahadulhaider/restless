package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/tidwall/gjson"
)

type ChainContext struct {
	Responses map[string]*model.Response
}

func NewChainContext() *ChainContext {
	return &ChainContext{Responses: make(map[string]*model.Response)}
}

func (ctx *ChainContext) StoreResponse(name string, resp *model.Response) {
	ctx.Responses[name] = resp
}

var varRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

func ResolveRequest(req *model.Request, vars map[string]string, chainCtx *ChainContext) (*model.Request, error) {
	resolved := *req

	resolved.URL = resolveString(req.URL, vars, chainCtx)

	resolved.Headers = make([]model.Header, len(req.Headers))
	for i, h := range req.Headers {
		resolved.Headers[i] = model.Header{
			Key:   h.Key,
			Value: resolveString(h.Value, vars, chainCtx),
		}
	}

	resolved.Body = resolveString(req.Body, vars, chainCtx)

	return &resolved, nil
}

func resolveString(s string, vars map[string]string, chainCtx *ChainContext) string {
	return varRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := match[2 : len(match)-2]

		if strings.HasPrefix(inner, "$") {
			return ResolveDynamicVars(match)
		}

		parts := strings.SplitN(inner, ".", 4)
		if len(parts) >= 3 && parts[1] == "response" && chainCtx != nil {
			name := parts[0]
			resp, ok := chainCtx.Responses[name]
			if !ok {
				return match
			}
			if parts[2] == "body" && len(parts) == 4 {
				result := gjson.GetBytes(resp.Body, parts[3])
				if !result.Exists() {
					return match
				}
				return result.String()
			}
			if parts[2] == "headers" && len(parts) == 4 {
				headerName := parts[3]
				for _, h := range resp.Headers {
					if strings.EqualFold(h.Key, headerName) {
						return h.Value
					}
				}
				return match
			}
		}

		if val, ok := vars[inner]; ok {
			return val
		}

		return match
	})
}

func ResolveChainVar(ctx *ChainContext, varRef string) (string, error) {
	parts := strings.SplitN(varRef, ".", 4)
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid chain variable reference: %q", varRef)
	}

	name := parts[0]
	resp, ok := ctx.Responses[name]
	if !ok {
		return "", fmt.Errorf("no response stored for request %q", name)
	}

	if parts[1] != "response" {
		return "", fmt.Errorf("invalid chain variable: expected 'response' segment")
	}

	if parts[2] == "body" && len(parts) == 4 {
		result := gjson.GetBytes(resp.Body, parts[3])
		if !result.Exists() {
			return "", fmt.Errorf("path %q not found in response body of %q", parts[3], name)
		}
		return result.String(), nil
	}

	if parts[2] == "headers" && len(parts) == 4 {
		for _, h := range resp.Headers {
			if strings.EqualFold(h.Key, parts[3]) {
				return h.Value, nil
			}
		}
		return "", fmt.Errorf("header %q not found in response from %q", parts[3], name)
	}

	return "", fmt.Errorf("invalid chain variable reference: %q", varRef)
}
