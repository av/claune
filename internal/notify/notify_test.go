package notify

import (
	"errors"
	"reflect"
	"testing"
)

type recordedCommand struct {
	started *bool
	err     error
}

func (c recordedCommand) Start() error {
	if c.started != nil {
		*c.started = true
	}
	return c.err
}

func TestSendSupportedPlatformsCommandContract(t *testing.T) {
	tests := []struct {
		name       string
		goos       string
		title      string
		message    string
		wantName   string
		wantArgs   []string
		startError error
	}{
		{
			name:     "darwin escapes quoted input in osascript payload",
			goos:     "darwin",
			title:    `Title with "quotes"`,
			message:  `Message with "quotes"`,
			wantName: "osascript",
			wantArgs: []string{"-e", `display notification "Message with \"quotes\"" with title "Title with \"quotes\""`},
		},
		{
			name:     "linux passes quoted input as escaped args",
			goos:     "linux",
			title:    `Title with "quotes"`,
			message:  `Message with "quotes"`,
			wantName: "notify-send",
			wantArgs: []string{`Title with \"quotes\"`, `Message with \"quotes\"`},
		},
		{
			name:     "windows escapes quoted input in powershell command",
			goos:     "windows",
			title:    `Title with "quotes"`,
			message:  `Message with "quotes"`,
			wantName: "powershell",
			wantArgs: []string{
				"-WindowStyle",
				"Hidden",
				"-Command",
				`[reflection.assembly]::loadwithpartialname("System.Windows.Forms") | Out-Null;$notify = new-object system.windows.forms.notifyicon;$notify.icon = [system.drawing.systemicons]::information;$notify.visible = $true;$notify.showballoontip(10, "Title with \"quotes\"", "Message with \"quotes\"", [system.windows.forms.tooltipicon]::None);Start-Sleep -Seconds 3`,
			},
			startError: errors.New("start failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalGOOS := currentGOOS
			originalFactory := commandFactory
			t.Cleanup(func() {
				currentGOOS = originalGOOS
				commandFactory = originalFactory
			})

			currentGOOS = tt.goos

			var gotName string
			var gotArgs []string
			started := false
			commandFactory = func(name string, args ...string) command {
				gotName = name
				gotArgs = append([]string(nil), args...)
				return recordedCommand{started: &started, err: tt.startError}
			}

			err := Send(tt.title, tt.message)
			if !errors.Is(err, tt.startError) {
				t.Fatalf("Send() error = %v, want %v", err, tt.startError)
			}

			if !started {
				t.Fatal("Send() did not start the captured command")
			}

			if gotName != tt.wantName {
				t.Fatalf("command name = %q, want %q", gotName, tt.wantName)
			}

			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Fatalf("command args = %#v, want %#v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestSendUnsupportedPlatformNoOp(t *testing.T) {
	originalGOOS := currentGOOS
	originalFactory := commandFactory
	t.Cleanup(func() {
		currentGOOS = originalGOOS
		commandFactory = originalFactory
	})

	currentGOOS = "plan9"

	called := false
	commandFactory = func(name string, args ...string) command {
		called = true
		return recordedCommand{}
	}

	err := Send("ignored", `message with "quotes"`)
	if err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}

	if called {
		t.Fatal("Send() invoked commandFactory for unsupported platform")
	}
}
