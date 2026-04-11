package notify

import (
	"os/exec"
	"runtime"
	"strings"
)

// Send sends a desktop notification with the given title and message.
// It spawns a background process so it doesn't block audio playback.
func Send(title, message string) error {
	// Escape quotes for shell safety
	title = strings.ReplaceAll(title, `"`, `\"`)
	message = strings.ReplaceAll(message, `"`, `\"`)

	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("osascript", "-e", `display notification "`+message+`" with title "`+title+`"`)
		return cmd.Start()
	case "linux":
		cmd := exec.Command("notify-send", title, message)
		return cmd.Start()
	case "windows":
		// A simple way to trigger a toast in Windows 10/11 using PowerShell
		psCmd := `[reflection.assembly]::loadwithpartialname("System.Windows.Forms") | Out-Null;` +
			`$notify = new-object system.windows.forms.notifyicon;` +
			`$notify.icon = [system.drawing.systemicons]::information;` +
			`$notify.visible = $true;` +
			`$notify.showballoontip(10, "` + title + `", "` + message + `", [system.windows.forms.tooltipicon]::None);` +
			`Start-Sleep -Seconds 3` // sleep so the script doesn't exit immediately before the balloon shows
		cmd := exec.Command("powershell", "-WindowStyle", "Hidden", "-Command", psCmd)
		return cmd.Start()
	}
	return nil
}
