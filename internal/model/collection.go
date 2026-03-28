package model

type Collection struct {
	RootDir string
	Files   []HTTPFile
}

type HTTPFile struct {
	Path     string
	Requests []Request
}
