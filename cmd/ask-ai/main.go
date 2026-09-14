package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
		fmt.Fprint(os.Stderr, formatRunError(err))
		return 1
	}

	configPath := flag.String("config", defaultPath, "path to config.json")
	fromClipboard := flag.Bool("from-clipboard", false, "use text already copied by an application integration")
	showVersion := flag.Bool("version", false, "print version")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: ask-ai [flags] [provider [current|NAME [HTTPS_URL]] | completion bash]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return 0
	}
	args := normalizeArgs(filepath.Base(os.Args[0]), flag.Args())
	if len(args) > 0 {
		switch args[0] {
		case "provider":
			return runProvider(*configPath, args[1:], os.Stdin, os.Stdout, os.Stderr)
		case "completion":
			if len(args) == 2 && args[1] == "bash" {
				fmt.Print(askai.BashCompletion())
				return 0
			}
			fmt.Fprintln(os.Stderr, "usage: ask-ai completion bash")
			return 2
		default:
			fmt.Fprintf(os.Stderr, "ask-ai: unknown command %q\n", args[0])
			return 2
		}
	}

	platform := desktop.New()
	config, err := askai.LoadConfig(*configPath)
	if err != nil {
		_ = platform.Notify("Ask AI", "Configuration is invalid. Check config.json.")
		fmt.Fprintln(os.Stderr, "ask-ai:", err)
		return 1
	}
	if *fromClipboard {
		err = askai.RunFromClipboard(config, platform)
	} else {
		err = askai.Run(config, platform)
	}
	if err != nil {
		fmt.Fprint(os.Stderr, formatRunError(err))
		return 1
	}
	return 0
}

func formatRunError(err error) string {
	if !errors.Is(err, askai.ErrNoSelection) {
		return fmt.Sprintf("ask-ai: %v\n", err)
	}
	return `ask-ai: no text selection was captured

Select text in an application and use your configured keybinding.

Provider commands:
  ask-ai provider                         # interactive picker
  ask-ai provider current
  ask-ai provider <chatgpt|claude|gemini|kimi>
  ask-ai provider custom <HTTPS_URL>

Run ask-ai -h to see all options.
`
}

func normalizeArgs(program string, args []string) []string {
	if program != "ask-ai-provider" {
		return args
	}
	return append([]string{"provider"}, args...)
}

func runProvider(configPath string, args []string, input io.Reader, output, errorOutput io.Writer) int {
	if len(args) == 0 {
		return promptForProvider(configPath, input, output, errorOutput)
	}
	if len(args) == 1 && args[0] == "current" {
		config, err := askai.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintln(errorOutput, "ask-ai provider:", err)
			return 1
		}
		fmt.Fprintln(output, config.Provider)
		return 0
	}
	if len(args) > 2 {
		fmt.Fprintln(errorOutput, "usage: ask-ai provider [current|NAME [HTTPS_URL]]")
		return 2
	}

	customURL := ""
	if len(args) == 2 {
		customURL = args[1]
	}
	if err := askai.SetProvider(configPath, args[0], customURL); err != nil {
		fmt.Fprintln(errorOutput, "ask-ai provider:", err)
		return 1
	}
	fmt.Fprintln(output, args[0])
	return 0
}

func promptForProvider(configPath string, input io.Reader, output, errorOutput io.Writer) int {
	config, err := askai.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintln(errorOutput, "ask-ai provider:", err)
		return 1
	}
	fmt.Fprintf(output, `Current provider: %s
Choose provider:
  1) ChatGPT
  2) Claude
  3) Gemini
  4) Kimi
  5) Custom URL
Selection [keep current]: `, config.Provider)

	scanner := bufio.NewScanner(input)
	if !scanner.Scan() {
		fmt.Fprintln(errorOutput, "ask-ai provider: no selection received")
		return 1
	}
	choice := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if choice == "" {
		fmt.Fprintf(output, "Provider unchanged: %s\n", config.Provider)
		return 0
	}
	providers := map[string]string{
		"1": "chatgpt", "chatgpt": "chatgpt",
		"2": "claude", "claude": "claude",
		"3": "gemini", "gemini": "gemini",
		"4": "kimi", "kimi": "kimi",
		"5": "custom", "custom": "custom",
	}
	provider, ok := providers[choice]
	if !ok {
		fmt.Fprintf(errorOutput, "ask-ai provider: invalid selection %q\n", choice)
		return 2
	}

	customURL := ""
	if provider == "custom" {
		fmt.Fprint(output, "Custom HTTPS URL: ")
		if !scanner.Scan() {
			fmt.Fprintln(errorOutput, "ask-ai provider: no custom URL received")
			return 1
		}
		customURL = strings.TrimSpace(scanner.Text())
	}
	if err := askai.SetProvider(configPath, provider, customURL); err != nil {
		fmt.Fprintln(errorOutput, "ask-ai provider:", err)
		return 1
	}
	fmt.Fprintf(output, "Provider set to %s\n", provider)
	return 0
}
