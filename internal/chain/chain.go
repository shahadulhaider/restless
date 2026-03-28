package chain

import (
	"fmt"
	"strings"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/tidwall/gjson"
)

type ChainContext struct {
	responses map[string]*model.Response
}

func NewChainContext() *ChainContext {
	return &ChainContext{responses: make(map[string]*model.Response)}
}

func (ctx *ChainContext) StoreResponse(name string, resp *model.Response) {
	ctx.responses[name] = resp
}

func (ctx *ChainContext) Resolve(varRef string) (string, error) {
	parts := strings.SplitN(varRef, ".", 4)
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid chain variable %q: expected requestName.response.body|headers.path", varRef)
	}

	name := parts[0]
	resp, ok := ctx.responses[name]
	if !ok {
		return "", fmt.Errorf("no response stored for request %q", name)
	}

	if parts[1] != "response" {
		return "", fmt.Errorf("invalid chain variable %q: expected 'response' as second segment", varRef)
	}

	switch parts[2] {
	case "body":
		result := gjson.GetBytes(resp.Body, parts[3])
		if !result.Exists() {
			return "", fmt.Errorf("path %q not found in response body of %q", parts[3], name)
		}
		return result.String(), nil

	case "headers":
		headerName := parts[3]
		for _, h := range resp.Headers {
			if strings.EqualFold(h.Key, headerName) {
				return h.Value, nil
			}
		}
		return "", fmt.Errorf("header %q not found in response from %q", headerName, name)

	default:
		return "", fmt.Errorf("invalid chain variable %q: expected 'body' or 'headers'", varRef)
	}
}

func (ctx *ChainContext) HasResponse(name string) bool {
	_, ok := ctx.responses[name]
	return ok
}

func (ctx *ChainContext) Names() []string {
	names := make([]string, 0, len(ctx.responses))
	for n := range ctx.responses {
		names = append(names, n)
	}
	return names
}
