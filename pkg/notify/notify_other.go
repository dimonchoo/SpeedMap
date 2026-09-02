//go:build !darwin

package notify

import (
	"fmt"
	"os/exec"
	"runtime"
)

func sendOSNotification(title, subtitle, message string) {
	if title == "" {
		title = "SpeedMap"
	}
	body := message
	if subtitle != "" {
		body = subtitle + " - " + message
	}

	go func() {
		switch runtime.GOOS {
		case "windows":
			psScript := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; $template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); $textNodes = $template.GetElementsByTagName("text"); $textNodes.Item(0).AppendChild($template.CreateTextNode(%q)) > $null; $textNodes.Item(1).AppendChild($template.CreateTextNode(%q)) > $null; $toast = [Windows.UI.Notifications.ToastNotification]::new($template); [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("SpeedMap").Show($toast);`, title, body)
			_ = exec.Command("powershell", "-NoProfile", "-Command", psScript).Run()
		default: // Linux
			_ = exec.Command("notify-send", title, body).Run()
		}
	}()
}
