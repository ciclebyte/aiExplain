package main

import (
	"embed"

	"github.com/ciclebyte/aiExplain/assets"
	"github.com/ciclebyte/aiExplain/cmd"
)

//go:embed resources/*
var resources embed.FS

func main() {
	assets.Resources = resources
	cmd.Execute()
}
