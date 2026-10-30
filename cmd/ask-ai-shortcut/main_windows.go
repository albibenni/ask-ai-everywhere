//go:build windows

// ask-ai-shortcut is built as a GUI executable. It starts the console CLI
// without creating a console, preserving the source application's focus.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"ask-ai-everywhere/internal/desktop"
	"golang.org/x/sys/windows"
)

func main() { os.Exit(run()) }

func run() int {
	name, _ := windows.UTF16PtrFromString(`Local\AskAIEverywhereShortcut`)
	mutex, err := windows.CreateMutex(nil, false, name)
	if mutex != 0 {
		defer windows.CloseHandle(mutex)
	}
	if err == windows.ERROR_ALREADY_EXISTS {
		return 0 // Ignore overlapping invocations while a draft is being prepared.
	}
	if err != nil {
		_ = desktop.New().Notify("Ask AI", "Could not start the shortcut: "+err.Error())
		return 1
	}
	executable, err := os.Executable()
	if err != nil {
		_ = desktop.New().Notify("Ask AI", "Could not locate the installed CLI.")
		return 1
	}
	command := exec.Command(filepath.Join(filepath.Dir(executable), "ask-ai.exe"), os.Args[1:]...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	// Hidden launches need a place for CLI errors; avoid an invisible failure.
	cache, err := os.UserCacheDir()
	if err != nil {
		_ = desktop.New().Notify("Ask AI", "Could not locate the shortcut log directory.")
		return 1
	}
	directory := filepath.Join(cache, "ask-ai")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		_ = desktop.New().Notify("Ask AI", "Could not create the shortcut log directory.")
		return 1
	}
	log, err := os.OpenFile(filepath.Join(directory, "shortcut.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		_ = desktop.New().Notify("Ask AI", "Could not create the shortcut log.")
		return 1
	}
	defer log.Close()
	command.Stdout, command.Stderr = log, log
	if err := command.Run(); err != nil {
		fmt.Fprintln(log, "shortcut:", err)
		_ = desktop.New().Notify("Ask AI", "The shortcut failed. See %LOCALAPPDATA%\\ask-ai\\shortcut.log, or run ask-ai in a terminal.")
		return 1
	}
	return 0
}
