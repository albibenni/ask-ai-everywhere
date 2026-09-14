package main

import (
	"flag"
	"fmt"
	"os"

	"ask-ai-everywhere/internal/askai"
	"ask-ai-everywhere/internal/desktop"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	defaultPath, err := askai.DefaultConfigPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ask-ai:", err)
		return 1
	}

	configPath := flag.String("config", defaultPath, "path to config.json")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return 0
	}

	platform := desktop.New()
	config, err := askai.LoadConfig(*configPath)
	if err != nil {
		_ = platform.Notify("Ask AI", "Configuration is invalid. Check config.json.")
		fmt.Fprintln(os.Stderr, "ask-ai:", err)
		return 1
	}
	if err := askai.Run(config, platform); err != nil {
		fmt.Fprintln(os.Stderr, "ask-ai:", err)
		return 1
	}
	return 0
}
