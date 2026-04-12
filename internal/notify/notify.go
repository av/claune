package notify

import (
	"os/exec"
	"runtime"
	"strings"
)

type command interface {
	Start() error
}

var currentGOOS = runtime.GOOS

var commandFactory = func(name string, args ...string) command {
	return exec.Command(name, args...)
}

// Send sends a desktop notification with the given title and message.
// It spawns a background process so it doesn't block audio playback.
func Send(title, message string) error {
	// Escape quotes for shell safety
	title = strings.ReplaceAll(title, `"`, `\"`)
	message = strings.ReplaceAll(message, `"`, `\"`)

	cmd, ok := notificationCommand(currentGOOS, title, message)
	if !ok {
		return nil
	}

	return cmd.Start()
}

func notificationCommand(goos, title, message string) (command, bool) {
	switch goos {
	case "darwin":
		return commandFactory("osascript", "-e", `display notification "`+message+`" with title "`+title+`"`), true
	case "linux":
		return commandFactory("notify-send", title, message), true
	case "windows":
		// A simple way to trigger a toast in Windows 10/11 using PowerShell
		psCmd := `[reflection.assembly]::loadwithpartialname("System.Windows.Forms") | Out-Null;` +
			`$notify = new-object system.windows.forms.notifyicon;` +
			`$notify.icon = [system.drawing.systemicons]::information;` +
			`$notify.visible = $true;` +
			`$notify.showballoontip(10, "` + title + `", "` + message + `", [system.windows.forms.tooltipicon]::None);` +
			`Start-Sleep -Seconds 3` // sleep so the script doesn't exit immediately before the balloon shows
		return commandFactory("powershell", "-WindowStyle", "Hidden", "-Command", psCmd), true
	}

	return nil, false
}
