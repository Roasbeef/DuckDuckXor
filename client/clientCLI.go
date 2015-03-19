package main

type CLI struct {
}

func NewCLI() *CLI {
	return &CLI{}
}

func (c *CLI) Start() {
	go c.ParseCommands()
}

func (c *CLI) ParseCommands() {
	for {

	}
}
