//go:build windows

package desktop

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"os/exec"
	"syscall"
	"time"
)

func (s *System) Notify(title, body string) error {
	toast := struct {
		XMLName xml.Name `xml:"toast"`
		Binding struct {
			Template string   `xml:"template,attr"`
			Text     []string `xml:"text"`
		} `xml:"visual>binding"`
	}{}
	toast.Binding.Template = "ToastGeneric"
	toast.Binding.Text = []string{title, body}
	payload, err := xml.Marshal(toast)
	if err != nil {
		return err
	}
	// Only base64 is interpolated into PowerShell; notification text is XML data.
	script := `$ErrorActionPreference = 'Stop'
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] > $null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml([Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('` + base64.StdEncoding.EncodeToString(payload) + `')))
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
$toast.Tag = 'ask-ai'
$toast.Group = 'ask-ai'
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Microsoft.Windows.PowerShell').Show($toast)`
	encoded, err := windowsClipboardText([]byte(script))
	if err != nil {
		return err
	}
	// -EncodedCommand uses UTF-16LE without a trailing NUL.
	commandText := base64.StdEncoding.EncodeToString(encoded[:len(encoded)-2])
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", commandText)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return command.Run()
}
