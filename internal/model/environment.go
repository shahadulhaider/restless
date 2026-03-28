package model

type Environment struct {
	Name      string
	Variables map[string]string
}

type EnvironmentFile struct {
	Shared       map[string]string
	Environments map[string]Environment
}
