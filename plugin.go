package main

import (
	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
	"strings"
)

func GoHighlight(vim *nvim.Nvim, args []string) (string, error) {
	return "", nil
}

func GenerateHightlight(vim *nvim.Nvim, args []string) (string, error) {
	neogo.ch <- struct{}{}
	return "", nil
}

func main() {
	plugin.Main(func(p *plugin.Plugin) error {
		p.HandleFunction(&plugin.FunctionOptions{Name: "GoHighlight"}, GoHighlight)
		p.HandleFunction(&plugin.FunctionOptions{Name: "goGenerateHightlight"}, GenerateHightlight)
		Serve(p.Nvim)
		return nil
	})
}

