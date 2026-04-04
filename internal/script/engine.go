package script

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dop251/goja"

	"github.com/shahadulhaider/restless/internal/model"
)

// ScriptContext holds the data available to scripts and collects modifications.
type ScriptContext struct {
	Request  *model.Request
	Response *model.Response // nil for pre-request
	EnvVars  map[string]string
	SetVars  map[string]string // variables set by setVar()
	LogOut   io.Writer         // log() output (default: os.Stderr)
}

// RunPreRequest executes a pre-request script that can modify the request.
func RunPreRequest(script string, ctx *ScriptContext) error {
	if script == "" {
		return nil
	}
	if ctx.SetVars == nil {
		ctx.SetVars = make(map[string]string)
	}
	if ctx.LogOut == nil {
		ctx.LogOut = os.Stderr
	}

	vm := goja.New()
	registerBuiltins(vm)

	// Expose request object
	reqObj := vm.NewObject()
	reqObj.Set("method", ctx.Request.Method)
	reqObj.Set("url", ctx.Request.URL)
	headers := make(map[string]string)
	for _, h := range ctx.Request.Headers {
		headers[h.Key] = h.Value
	}
	reqObj.Set("headers", headers)
	reqObj.Set("body", ctx.Request.Body)
	vm.Set("request", reqObj)

	// Expose env
	envObj := make(map[string]string)
	for k, v := range ctx.EnvVars {
		envObj[k] = v
	}
	vm.Set("env", envObj)

	// Mutating functions
	vm.Set("setHeader", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		value := call.Argument(1).String()
		// Update or add header
		found := false
		for i, h := range ctx.Request.Headers {
			if strings.EqualFold(h.Key, key) {
				ctx.Request.Headers[i].Value = value
				found = true
				break
			}
		}
		if !found {
			ctx.Request.Headers = append(ctx.Request.Headers, model.Header{Key: key, Value: value})
		}
		return goja.Undefined()
	})

	vm.Set("removeHeader", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		filtered := ctx.Request.Headers[:0]
		for _, h := range ctx.Request.Headers {
			if !strings.EqualFold(h.Key, key) {
				filtered = append(filtered, h)
			}
		}
		ctx.Request.Headers = filtered
		return goja.Undefined()
	})

	vm.Set("setBody", func(call goja.FunctionCall) goja.Value {
		ctx.Request.Body = call.Argument(0).String()
		return goja.Undefined()
	})

	vm.Set("setUrl", func(call goja.FunctionCall) goja.Value {
		ctx.Request.URL = call.Argument(0).String()
		return goja.Undefined()
	})

	vm.Set("setVar", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		value := call.Argument(1).String()
		ctx.SetVars[key] = value
		return goja.Undefined()
	})

	vm.Set("log", func(call goja.FunctionCall) goja.Value {
		msg := call.Argument(0).String()
		fmt.Fprintln(ctx.LogOut, "[script]", msg)
		return goja.Undefined()
	})

	return runWithTimeout(vm, script, 5*time.Second)
}

// RunPostResponse executes a post-response script that can read the response and set variables.
func RunPostResponse(script string, ctx *ScriptContext) error {
	if script == "" {
		return nil
	}
	if ctx.SetVars == nil {
		ctx.SetVars = make(map[string]string)
	}
	if ctx.LogOut == nil {
		ctx.LogOut = os.Stderr
	}

	vm := goja.New()
	registerBuiltins(vm)

	// Expose request (read-only in post-response)
	reqObj := vm.NewObject()
	reqObj.Set("method", ctx.Request.Method)
	reqObj.Set("url", ctx.Request.URL)
	headers := make(map[string]string)
	for _, h := range ctx.Request.Headers {
		headers[h.Key] = h.Value
	}
	reqObj.Set("headers", headers)
	reqObj.Set("body", ctx.Request.Body)
	vm.Set("request", reqObj)

	// Expose response
	if ctx.Response != nil {
		respObj := vm.NewObject()
		respObj.Set("status", ctx.Response.StatusCode)

		respHeaders := make(map[string]string)
		for _, h := range ctx.Response.Headers {
			respHeaders[h.Key] = h.Value
		}
		respObj.Set("headers", respHeaders)
		respObj.Set("time", ctx.Response.Timing.Total.Milliseconds())

		// Parse body as JSON if possible, otherwise string
		var bodyVal interface{}
		if err := json.Unmarshal(ctx.Response.Body, &bodyVal); err == nil {
			respObj.Set("body", bodyVal)
		} else {
			respObj.Set("body", string(ctx.Response.Body))
		}

		vm.Set("response", respObj)
	}

	// Expose env
	envObj := make(map[string]string)
	for k, v := range ctx.EnvVars {
		envObj[k] = v
	}
	vm.Set("env", envObj)

	// setVar is available in post-response
	vm.Set("setVar", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		value := call.Argument(1).String()
		ctx.SetVars[key] = value
		return goja.Undefined()
	})

	vm.Set("log", func(call goja.FunctionCall) goja.Value {
		msg := call.Argument(0).String()
		fmt.Fprintln(ctx.LogOut, "[script]", msg)
		return goja.Undefined()
	})

	// Mutating request functions are no-ops in post-response
	noop := func(call goja.FunctionCall) goja.Value { return goja.Undefined() }
	vm.Set("setHeader", noop)
	vm.Set("removeHeader", noop)
	vm.Set("setBody", noop)
	vm.Set("setUrl", noop)

	return runWithTimeout(vm, script, 5*time.Second)
}

func runWithTimeout(vm *goja.Runtime, script string, timeout time.Duration) error {
	// Set up timeout via interrupt
	timer := time.AfterFunc(timeout, func() {
		vm.Interrupt("script timeout exceeded")
	})
	defer timer.Stop()

	_, err := vm.RunString(script)
	if err != nil {
		return fmt.Errorf("script error: %w", err)
	}
	return nil
}
