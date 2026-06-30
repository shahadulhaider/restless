package main

type GreetService struct {
	rootDir string
}

func (g *GreetService) Greet(name string) string {
	return "Hello " + name + " from Restless!"
}

func (g *GreetService) GetRootDir() string {
	return g.rootDir
}
