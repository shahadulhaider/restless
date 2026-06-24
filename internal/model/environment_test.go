package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentZeroValue(t *testing.T) {
	var e Environment

	assert.Equal(t, "", e.Name)
	assert.Nil(t, e.Variables)
	assert.Empty(t, e.Variables)
}

func TestEnvironmentNilMapRead(t *testing.T) {
	var e Environment

	assert.Equal(t, "", e.Variables["anything"])

	v, ok := e.Variables["anything"]
	assert.False(t, ok)
	assert.Equal(t, "", v)
	assert.Len(t, e.Variables, 0)
}

func TestEnvironmentNilMapWritePanics(t *testing.T) {
	var e Environment

	assert.Panics(t, func() {
		e.Variables["x"] = "y"
	})
}

func TestEnvironmentVariables(t *testing.T) {
	e := Environment{
		Name: "dev",
		Variables: map[string]string{
			"baseUrl": "http://localhost:8000",
			"token":   "abc",
		},
	}

	assert.Equal(t, "dev", e.Name)
	assert.Len(t, e.Variables, 2)
	assert.Equal(t, "http://localhost:8000", e.Variables["baseUrl"])
	assert.Equal(t, "abc", e.Variables["token"])

	e.Variables["token"] = "xyz"
	assert.Equal(t, "xyz", e.Variables["token"])

	e.Variables["extra"] = "1"
	assert.Len(t, e.Variables, 3)

	delete(e.Variables, "extra")
	assert.Len(t, e.Variables, 2)
	_, ok := e.Variables["extra"]
	assert.False(t, ok)
}

func TestEnvironmentConstruction(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		wantName string
		wantLen  int
	}{
		{"named empty map", Environment{Name: "prod", Variables: map[string]string{}}, "prod", 0},
		{"named with vars", Environment{Name: "staging", Variables: map[string]string{"a": "1", "b": "2"}}, "staging", 2},
		{"unnamed", Environment{Variables: map[string]string{"k": "v"}}, "", 1},
		{"zero", Environment{}, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantName, tt.env.Name)
			assert.Len(t, tt.env.Variables, tt.wantLen)
		})
	}
}

func TestEnvironmentFileZeroValue(t *testing.T) {
	var ef EnvironmentFile

	assert.Nil(t, ef.Shared)
	assert.Nil(t, ef.Environments)
	assert.Empty(t, ef.Shared)
	assert.Empty(t, ef.Environments)
}

func TestEnvironmentFileConstruction(t *testing.T) {
	ef := EnvironmentFile{
		Shared: map[string]string{"apiKey": "shared-key"},
		Environments: map[string]Environment{
			"dev":  {Name: "dev", Variables: map[string]string{"baseUrl": "http://localhost"}},
			"prod": {Name: "prod", Variables: map[string]string{"baseUrl": "https://api.example.com"}},
		},
	}

	assert.Len(t, ef.Shared, 1)
	assert.Equal(t, "shared-key", ef.Shared["apiKey"])

	assert.Len(t, ef.Environments, 2)
	assert.Equal(t, "dev", ef.Environments["dev"].Name)
	assert.Equal(t, "http://localhost", ef.Environments["dev"].Variables["baseUrl"])
	assert.Equal(t, "prod", ef.Environments["prod"].Name)
	assert.Equal(t, "https://api.example.com", ef.Environments["prod"].Variables["baseUrl"])
}

func TestEnvironmentFileUnknownEnv(t *testing.T) {
	ef := EnvironmentFile{
		Environments: map[string]Environment{
			"dev": {Name: "dev"},
		},
	}

	missing, ok := ef.Environments["staging"]
	assert.False(t, ok)
	assert.Equal(t, Environment{}, missing)
	assert.Equal(t, "", missing.Name)
	assert.Nil(t, missing.Variables)
}

func TestEnvironmentFileDeepEqual(t *testing.T) {
	build := func() EnvironmentFile {
		return EnvironmentFile{
			Shared: map[string]string{"region": "us"},
			Environments: map[string]Environment{
				"dev": {Name: "dev", Variables: map[string]string{"baseUrl": "http://localhost"}},
			},
		}
	}

	a := build()
	b := build()
	assert.Equal(t, a, b)

	b.Environments["dev"].Variables["baseUrl"] = "http://changed"
	assert.NotEqual(t, a, b)
}
