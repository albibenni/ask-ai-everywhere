//go:build darwin

package desktop

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"ask-ai-everywhere/internal/askai"
)

type System struct{}

func New() *System { return &System{} }

func (s *System) ReadClipboard() (askai.ClipboardItem, error) {
	output, err := runJavaScript(macosClipboardReadScript)
	if err != nil {
		return askai.ClipboardItem{}, err
	}
	return parseMacOSClipboard(output)
}

func (s *System) WriteClipboard(item askai.ClipboardItem) error {
	if !strings.HasPrefix(item.MIMEType, "image/") {
		command := exec.Command("pbcopy")
		command.Stdin = bytes.NewReader(item.Data)
		return command.Run()
	}

	pasteboardType := macOSPasteboardType(item.MIMEType)
	if pasteboardType == "" {
		return fmt.Errorf("unsupported macOS image MIME type %q", item.MIMEType)
	}
	command := exec.Command("osascript", "-l", "JavaScript", "-e", macosClipboardWriteScript, pasteboardType)
	command.Stdin = bytes.NewReader([]byte(base64.StdEncoding.EncodeToString(item.Data)))
	return command.Run()
}

func (s *System) CopySelection() error {
	return runAppleScript(`tell application "System Events" to keystroke "c" using command down`)
}

func (s *System) OpenURL(url string) error {
	return exec.Command("open", url).Run()
}

func (s *System) ReplaceDraft(provider, _ string) error {
	composerFocus := ""
	if provider == "kimi" {
		composerFocus = `keystroke "k" using command down
delay 0.1
`
	}
	return runAppleScript(`tell application "System Events"
` + composerFocus + `keystroke "a" using command down
delay 0.05
key code 51
delay 0.05
keystroke "v" using command down
end tell`)
}

func (s *System) Notify(title, body string) error {
	script := `on run argv
display notification (item 2 of argv) with title (item 1 of argv)
end run`
	return exec.Command("osascript", "-e", script, title, body).Run()
}

func (s *System) Sleep(duration time.Duration) { time.Sleep(duration) }

func runAppleScript(script string) error {
	return exec.Command("osascript", "-e", script).Run()
}

func runJavaScript(script string) ([]byte, error) {
	return exec.Command("osascript", "-l", "JavaScript", "-e", script).Output()
}

const macosClipboardReadScript = `
ObjC.import('AppKit')

function encode(mimeType, data) {
    if (data === null || data === undefined || ObjC.unwrap(data) === undefined) return null
    return mimeType + '\n' + ObjC.unwrap(data.base64EncodedStringWithOptions(0))
}

function run() {
    var pasteboard = $.NSPasteboard.generalPasteboard
    var image = encode('image/png', pasteboard.dataForType($.NSPasteboardTypePNG)) ||
        encode('image/tiff', pasteboard.dataForType($.NSPasteboardTypeTIFF)) ||
        encode('image/jpeg', pasteboard.dataForType($('public.jpeg')))
    if (image !== null) return image

    var text = pasteboard.stringForType($.NSPasteboardTypeString)
    if (text === null) {
        return 'text/plain;charset=utf-8\n'
    }
    return encode('text/plain;charset=utf-8', $(text).dataUsingEncoding($.NSUTF8StringEncoding))
}
`

const macosClipboardWriteScript = `
ObjC.import('AppKit')

function run(argv) {
    const encoded = $.NSString.alloc.initWithDataEncoding(
        $.NSFileHandle.fileHandleWithStandardInput.readDataToEndOfFile,
        $.NSUTF8StringEncoding,
    )
    const data = $.NSData.alloc.initWithBase64EncodedStringOptions(encoded, 0)
    if (data === null) throw new Error('invalid image data')

    const pasteboard = $.NSPasteboard.generalPasteboard
    pasteboard.clearContents
    if (!pasteboard.setDataForType(data, $(argv[0]))) {
        throw new Error('could not write image to pasteboard')
    }
}
`
