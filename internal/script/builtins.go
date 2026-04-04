package script

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	crand "crypto/rand"

	"github.com/dop251/goja"
)

// registerBuiltins adds utility functions to the JS runtime.
func registerBuiltins(vm *goja.Runtime) {
	vm.Set("base64Encode", func(call goja.FunctionCall) goja.Value {
		s := call.Argument(0).String()
		return vm.ToValue(base64.StdEncoding.EncodeToString([]byte(s)))
	})

	vm.Set("base64Decode", func(call goja.FunctionCall) goja.Value {
		s := call.Argument(0).String()
		decoded, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(string(decoded))
	})

	vm.Set("hmac_sha256", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		data := call.Argument(1).String()
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(data))
		return vm.ToValue(hex.EncodeToString(mac.Sum(nil)))
	})

	vm.Set("sha256", func(call goja.FunctionCall) goja.Value {
		data := call.Argument(0).String()
		hash := sha256.Sum256([]byte(data))
		return vm.ToValue(hex.EncodeToString(hash[:]))
	})

	vm.Set("md5", func(call goja.FunctionCall) goja.Value {
		data := call.Argument(0).String()
		hash := md5.Sum([]byte(data))
		return vm.ToValue(hex.EncodeToString(hash[:]))
	})

	vm.Set("uuid", func(call goja.FunctionCall) goja.Value {
		b := make([]byte, 16)
		_, _ = crand.Read(b)
		b[6] = (b[6] & 0x0f) | 0x40
		b[8] = (b[8] & 0x3f) | 0x80
		id := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
		return vm.ToValue(id)
	})

	vm.Set("timestamp", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(time.Now().Unix())
	})

	vm.Set("isoTimestamp", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(time.Now().UTC().Format(time.RFC3339))
	})
}
