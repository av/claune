package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/everlier/claune/internal/ai"
	"github.com/everlier/claune/internal/audio"
	"github.com/everlier/claune/internal/circus"
	"github.com/everlier/claune/internal/config"
)

const playUsage = "Usage: claune play <event>\n       claune play <event> <tool-name> <tool-input>"

var clauneSubcommands = map[string]bool{
	"play":          true,
	"install":       true,
	"uninstall":     true,
	"status":        true,
	"test-sounds":   true,
	"help":          true,
	"--help":        true,
	"-h":            true,
	"config":        true,
	"import-circus": true,
	"analyze-log":   true,
	"automap":       true,
	"analyze-resp":  true,
	"website":       true,
	"skins":         true,
	"geocities":     true,
}

func Run(args []string) error {
	if len(args) == 0 || !clauneSubcommands[args[0]] {
		runPassthrough(args)
		return nil
	}

	if len(args) >= 2 && (args[1] == "--help" || args[1] == "-h") {
		printCommandUsage(args[0])
		return nil
	}

	switch args[0] {
	case "help", "--help", "-h":
		if len(args) == 2 {
			printCommandUsage(args[1])
			return nil
		}
		ensureExactArgs(args, 1, "claune: help does not accept additional arguments", "Usage: claune help")
		printUsage()
		return nil
	case "install":
		ensureExactArgs(args, 1, "claune: install does not accept additional arguments", "Usage: claune install")
		return installHooks()
	case "uninstall":
		ensureExactArgs(args, 1, "claune: uninstall does not accept additional arguments", "Usage: claune uninstall")
		return uninstallHooks()
	case "website":
		ensureExactArgs(args, 1, "claune: website does not accept additional arguments", "Usage: claune website")
		url := "https://av.github.io/claune/"
		fmt.Printf("\033[36mVISIT THE OFFICIAL CYBER PORTAL:\033[0m %s\n", url)
		openBrowser(url)
		return nil
	case "skins":
		ensureExactArgs(args, 1, "claune: skins does not accept additional arguments", "Usage: claune skins")
		for i := 0; i <= 10; i++ {
			bars := strings.Repeat("=", i)
			spaces := strings.Repeat(" ", 10-i)
			percent := i * 10
			fmt.Printf("\rDownloading Matrix_Green_Theme.wsz [%s%s] %d%%", bars, spaces, percent)
			time.Sleep(200 * time.Millisecond)
		}
		fmt.Println()
		time.Sleep(300 * time.Millisecond)
		fmt.Println("\033[31m[!] Winamp.exe has encountered a fatal exception 0xDEADBEEF\033[0m")
		return nil
	case "geocities":
		ensureExactArgs(args, 1, "claune: geocities does not accept additional arguments", "Usage: claune geocities")
		fmt.Println("Connecting to ftp.geocities.com on port 21...")
		time.Sleep(800 * time.Millisecond)
		fmt.Println("Connected. Waiting for welcome message...")
		time.Sleep(600 * time.Millisecond)
		fmt.Println("220 ftp.geocities.com FTP server ready.")
		time.Sleep(400 * time.Millisecond)
		fmt.Println("USER xX_Everlier_Xx")
		time.Sleep(500 * time.Millisecond)
		fmt.Println("331 Password required for xX_Everlier_Xx.")
		time.Sleep(400 * time.Millisecond)
		fmt.Println("PASS ********")
		time.Sleep(1200 * time.Millisecond)
		fmt.Println("230 User xX_Everlier_Xx logged in.")
		time.Sleep(300 * time.Millisecond)
		fmt.Println("TYPE I")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("200 Type set to I.")
		time.Sleep(300 * time.Millisecond)
		fmt.Println("PASV")
		time.Sleep(400 * time.Millisecond)
		fmt.Println("227 Entering Passive Mode (209,1,224,42,14,17).")
		time.Sleep(500 * time.Millisecond)
		fmt.Println("STOR index.html")
		time.Sleep(600 * time.Millisecond)
		fmt.Println("150 Opening BINARY mode data connection for index.html.")
		time.Sleep(1500 * time.Millisecond)
		fmt.Println("226 Transfer complete.")
		time.Sleep(300 * time.Millisecond)
		fmt.Println("QUIT")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("221 Goodbye.")
		time.Sleep(500 * time.Millisecond)
		fmt.Println("\033[35m~*~ GEOCITIES UPLOAD COMPLETE ~*~\033[0m")
		return nil
	}

	validateManagementArgs(args)

	c, err := loadCommandConfig(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "claune: error loading config: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "play":
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			printCommandUsage("play")
			return nil
		}
		if len(args) == 4 {
			event, err := ai.AnalyzeToolIntent(args[1], args[2], args[3], c)
			if err != nil && c.AI.Enabled {
				fmt.Fprintf(os.Stderr, "⚠️ AI Semantic Audio Error: %v\n", err)
			}
			if err := audio.PlaySound(event, true, c); err != nil {
				fmt.Fprintf(os.Stderr, "Error playing sound: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := audio.PlaySound(args[1], true, c); err != nil {
				fmt.Fprintf(os.Stderr, "Error playing sound: %v\n", err)
				os.Exit(1)
			}
		}
	case "status":
		showStatus(c)
	case "test-sounds":
		testSounds(c)
	case "config":
		prompt := strings.Join(args[1:], " ")
		if err := ai.HandleNaturalLanguageConfig(prompt, &c); err != nil {
			return fmt.Errorf("AI config failed: %w", err)
		}
		fmt.Println("Config updated successfully via AI")
	case "import-circus":
		url := args[1]
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}
		filename := args[2]
		cachedPath := filepath.Join(audio.SoundCacheDir(), filename)
		if err := circus.ImportMemeSound(url, filename); err != nil {
			fmt.Fprintf(os.Stderr, "Import failed: %v\n", err)
			os.Exit(1)
		} else {
			var event string
			if len(args) == 4 {
				event = args[3]
			} else {
				guessed, err := ai.GuessEventForSound(url, filename, c)
				if err != nil {
					fmt.Printf("Imported %s to %s, but could not map it to an event automatically.\n", filename, cachedPath)
					fmt.Fprintf(os.Stderr, "AI mapping failed: %v. Please rerun with an explicit event to update ~/.config/claune/config.json.\n", err)
					os.Exit(2)
				}
				event = guessed
			}

			if c.Sounds == nil {
				c.Sounds = make(map[string]config.EventSoundConfig)
			}

			// Keep existing config if it exists, just append/overwrite
			eventCfg := c.Sounds[event]
			eventCfg.Paths = append(eventCfg.Paths, cachedPath)
			c.Sounds[event] = eventCfg

			if err := config.Save(c); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to update config: %v\n", err)
				fmt.Fprintf(os.Stderr, "%s was downloaded to %s, but claune could not update ~/.config/claune/config.json.\n", filename, cachedPath)
				os.Exit(1)
			} else {
				fmt.Printf("Imported %s and mapped it to event %s\n", filename, event)
			}
		}
	case "analyze-log":
		var logText string
		if len(args) > 1 {
			logText = strings.Join(args[1:], " ")
		} else {
			logText = mustReadStdin("analyze-log")
		}
		if !utf8.ValidString(logText) {
			fmt.Fprintln(os.Stderr, "claune: error: input data is not valid UTF-8 text")
			os.Exit(1)
		}
		circus.AnalyzeLogSentiment(logText, c, true)
	case "automap":
		dir := args[1]
		mapping, err := ai.AutoMapSounds(dir, &c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Automap failed: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println("Sounds mapped successfully:")
			for event, cfg := range mapping {
				if len(cfg.Paths) > 0 {
					fmt.Printf("  - %s mapped to: %s\n", event, cfg.Paths[0])
				}
			}
		}
	case "analyze-resp":
		var respText string
		if len(args) > 1 {
			respText = strings.Join(args[1:], " ")
		} else {
			respText = mustReadStdin("analyze-resp")
		}
		if !utf8.ValidString(respText) {
			fmt.Fprintln(os.Stderr, "claune: error: input data is not valid UTF-8 text")
			os.Exit(1)
		}
		event, strategy, err := ai.AnalyzeResponseSentiment(respText, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Analyze response sentiment failed: %v\n", err)
			os.Exit(1)
		} else if event != "" {
			if err := audio.PlaySoundWithStrategy(event, strategy, true, c); err != nil {
				fmt.Fprintf(os.Stderr, "Error playing sound: %v\n", err)
				os.Exit(1)
			}
		}
	}
	return nil
}

func ensureExactArgs(args []string, expected int, message string, usage string) {
	if len(args) != expected {
		exitUsageError(message, usage)
	}
}

func validateManagementArgs(args []string) {
	switch args[0] {
	case "status":
		ensureExactArgs(args, 1, "claune: status does not accept additional arguments", "Usage: claune status")
	case "play":
		if err := validatePlayArgs(args); err != nil {
			exitUsageError(err.Error(), playUsage)
		}
	case "test-sounds":
		ensureExactArgs(args, 1, "claune: test-sounds does not accept additional arguments", "Usage: claune test-sounds")
	case "config":
		if len(args) <= 1 {
			exitUsageError("claune: config requires a natural language prompt", "Usage: claune config <natural language prompt>\nExamples:\n  claune config \"mute sound\"\n  claune config \"set volume to 50%\"")
		}
	case "automap":
		switch len(args) {
		case 1:
			exitUsageError("claune: automap requires a directory", "Usage: claune automap <directory>")
		case 2:
			return
		default:
			exitUsageError("claune: automap does not accept additional arguments", "Usage: claune automap <directory>")
		}
	case "import-circus":
		switch len(args) {
		case 1, 2:
			exitUsageError("claune: import-circus requires a URL and name", "Usage: claune import-circus <url> <name> [event]")
		case 3, 4:
			return
		default:
			exitUsageError("claune: import-circus does not accept additional arguments", "Usage: claune import-circus <url> <name> [event]")
		}
	case "analyze-log":
		return
	case "analyze-resp":
		return
	case "website":
		ensureExactArgs(args, 1, "claune: website does not accept additional arguments", "Usage: claune website")
	case "skins":
		ensureExactArgs(args, 1, "claune: skins does not accept additional arguments", "Usage: claune skins")
	case "geocities":
		ensureExactArgs(args, 1, "claune: geocities does not accept additional arguments", "Usage: claune geocities")
	}
}

func validatePlayArgs(args []string) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("claune: play requires an event")
	case 2, 4:
		return nil
	case 3:
		return fmt.Errorf("claune: play accepts either <event> or <event> <tool-name> <tool-input>")
	default:
		return fmt.Errorf("claune: play does not accept additional arguments")
	}
}

func exitUsageError(message string, usage string) {
	fmt.Fprintln(os.Stderr, message)
	fmt.Fprintln(os.Stderr, usage)
	os.Exit(1)
}

func mustReadStdin(command string) string {
	info, err := os.Stdin.Stat()
	if err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
		fmt.Fprintf(os.Stderr, "claune: %s requires piped input or direct string arguments\n", command)
		os.Exit(1)
	}

	const maxBytes = 10 * 1024 * 1024 // 10MB limit
	const chunkSize = 32 * 1024       // 32KB head and 32KB tail

	// If it's a regular file and larger than our limit, seek to avoid reading the whole file
	if err == nil && info.Mode().IsRegular() && info.Size() > maxBytes {
		head := make([]byte, chunkSize)
		n, err := os.Stdin.Read(head)
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "claune: failed to read head for %s: %v\n", command, err)
			os.Exit(1)
		}
		headStr := string(head[:n])

		tail := make([]byte, chunkSize)
		_, err = os.Stdin.Seek(int64(-chunkSize), io.SeekEnd)
		if err == nil {
			n, err = io.ReadFull(os.Stdin, tail)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				fmt.Fprintf(os.Stderr, "claune: failed to read tail for %s: %v\n", command, err)
				os.Exit(1)
			}
			return headStr + "\n\n... [truncated massive file] ...\n\n" + string(tail[:n])
		}
		// If seek fails for some reason, we fall through to streaming
		_, _ = os.Stdin.Seek(0, io.SeekStart)
	}

	// Limit to 10MB to prevent endless piping DoS, but stream to avoid memory spikes
	limitReader := io.LimitReader(os.Stdin, maxBytes)

	head := make([]byte, 0, chunkSize)
	tail := make([]byte, chunkSize)
	tailPos := 0
	tailBytes := 0

	buf := make([]byte, 4096)
	for {
		n, err := limitReader.Read(buf)
		if n > 0 {
			chunk := buf[:n]

			// Fill head first
			if len(head) < chunkSize {
				space := chunkSize - len(head)
				if n <= space {
					head = append(head, chunk...)
					continue
				} else {
					head = append(head, chunk[:space]...)
					chunk = chunk[space:]
				}
			}

			// Put the rest in tail circular buffer
			for len(chunk) > 0 {
				space := chunkSize - tailPos
				if len(chunk) < space {
					space = len(chunk)
				}
				copy(tail[tailPos:], chunk[:space])
				tailPos += space
				tailBytes += space
				chunk = chunk[space:]
				if tailPos == chunkSize {
					tailPos = 0
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "claune: failed to read stdin for %s: %v\n", command, err)
			os.Exit(1)
		}
	}

	var result strings.Builder
	
	expectedLen := len(head)
	if tailBytes > chunkSize {
		expectedLen += len("\n\n... [truncated mid-stream] ...\n\n") + chunkSize
	} else if tailBytes > 0 {
		expectedLen += tailBytes
	}
	result.Grow(expectedLen)
	
	result.Write(head)

	if tailBytes > chunkSize {
		result.WriteString("\n\n... [truncated mid-stream] ...\n\n")
		result.Write(tail[tailPos:])
		result.Write(tail[:tailPos])
	} else if tailBytes > 0 {
		result.Write(tail[:tailBytes])
	}

	return result.String()
}

func loadCommandConfig(command string) (config.ClauneConfig, error) {
	c, err := config.Load()
	if err == nil {
		return c, nil
	}

	if command == "config" && config.IsInvalidConfigError(err) {
		fmt.Fprintf(os.Stderr, "claune: warning: invalid config, starting from defaults: %v\n", err)
		return c, nil
	}

	return c, err
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		// Ignore error
	}
}

func showStatus(c config.ClauneConfig) {
	if hooksInstalled() {
		fmt.Println("Installed — claune hooks are active in Claude Code.")
	} else {
		fmt.Println("Not installed — run 'claune install' to add sound hooks.")
	}

	if c.ShouldMute() {
		fmt.Println("Sound: muted")
	} else {
		fmt.Printf("Volume: %.0f%%\n", c.GetVolume()*100)
	}
}

func testSounds(c config.ClauneConfig) {
	if c.ShouldMute() {
		return
	}
	fmt.Println("Testing all sounds...")
	hasError := false
	for _, event := range []string{"cli:start", "tool:start", "tool:success", "tool:error", "cli:done", "build:success", "test:fail", "panic", "warn"} {
		fmt.Printf("  %s ", event)
		if err := audio.PlaySound(event, true, c); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			hasError = true
		} else {
			fmt.Println("OK")
		}
	}
	if hasError {
		os.Exit(1)
	}
}

func printCommandUsage(cmd string) {
	switch cmd {
	case "play":
		fmt.Fprintln(os.Stderr, "Usage: claune play <event>\n       claune play <event> <tool-name> <tool-input>")
		fmt.Fprintln(os.Stderr, "\nPlays a sound associated with the given event.")
		fmt.Fprintln(os.Stderr, "If a tool name and input are provided, AI semantic audio mapping is used.")
		fmt.Fprintln(os.Stderr, "Audio limits: Max 50MB or 5 minutes of decompressed audio.")
		fmt.Fprintln(os.Stderr, "Linux fallback: Tries paplay, pw-play, then aplay.")
	case "install":
		fmt.Fprintln(os.Stderr, "Usage: claune install")
		fmt.Fprintln(os.Stderr, "\nInstalls sound hooks into Claude Code settings.")
	case "uninstall":
		fmt.Fprintln(os.Stderr, "Usage: claune uninstall")
		fmt.Fprintln(os.Stderr, "\nRemoves sound hooks from Claude Code settings.")
	case "status":
		fmt.Fprintln(os.Stderr, "Usage: claune status")
		fmt.Fprintln(os.Stderr, "\nShows whether hooks are installed and current volume/mute status.")
	case "test-sounds":
		fmt.Fprintln(os.Stderr, "Usage: claune test-sounds")
		fmt.Fprintln(os.Stderr, "\nPlays all available sounds sequentially to verify audio works.")
	case "config":
		fmt.Fprintln(os.Stderr, "Usage: claune config <natural language prompt>")
		fmt.Fprintln(os.Stderr, "\nExamples:\n  claune config \"mute sound\"\n  claune config \"set volume to 50%\"")
		fmt.Fprintln(os.Stderr, "\nUses AI to update the configuration. Falls back to default limits (2048 tokens max).")
	case "automap":
		fmt.Fprintln(os.Stderr, "Usage: claune automap <directory>")
		fmt.Fprintln(os.Stderr, "\nAutomatically maps sound files in the given directory to events.")
		fmt.Fprintln(os.Stderr, "Uses AI for semantic mapping, falls back to regex matching if AI is unavailable.")
		fmt.Fprintln(os.Stderr, "Limits: Scans up to 500 audio files per directory.")
	case "import-circus":
		fmt.Fprintln(os.Stderr, "Usage: claune import-circus <url> <name> [event]")
		fmt.Fprintln(os.Stderr, "\nDownloads a meme sound from the given URL and maps it to an event.")
		fmt.Fprintln(os.Stderr, "Limits: 50MB max file size. 30-second timeout.")
	case "analyze-log":
		fmt.Fprintln(os.Stderr, "Usage: claune analyze-log [log text]")
		fmt.Fprintln(os.Stderr, "\nAnalyzes log text for sentiment and plays an appropriate sound.")
		fmt.Fprintln(os.Stderr, "Reads from stdin if no text is provided. Truncates inputs larger than 64KB (10MB hard limit).")
	case "analyze-resp":
		fmt.Fprintln(os.Stderr, "Usage: claune analyze-resp [response text]")
		fmt.Fprintln(os.Stderr, "\nAnalyzes AI response text and overrides playback strategy dynamically.")
		fmt.Fprintln(os.Stderr, "Reads from stdin if no text is provided. Truncates inputs larger than 64KB (10MB hard limit).")
	case "website":
		fmt.Fprintln(os.Stderr, "Usage: claune website")
		fmt.Fprintln(os.Stderr, "\nLaunch the official cyber portal in your default web browser.")
	case "skins":
		fmt.Fprintln(os.Stderr, "Usage: claune skins")
		fmt.Fprintln(os.Stderr, "\nDownload custom Winamp 2.95 skins for Claune.")
	case "geocities":
		fmt.Fprintln(os.Stderr, "Usage: claune geocities")
		fmt.Fprintln(os.Stderr, "\nRun a fake 90s-era WS_FTP terminal log to GeoCities.")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: claune [claude-args...]    Run Claude Code with sound effects
       claune <command>            Run a claune management command

Passthrough mode (default):
  claune                     Start Claude Code interactively with sounds

Management commands:
  install       Install sound hooks into Claude Code settings
  uninstall     Remove sound hooks from Claude Code settings
  status        Show whether hooks are installed
  play <event>  Play a sound for an event
  play <event> <tool-name> <tool-input>
                 Play a sound using semantic tool context
  test-sounds   Play all sounds to verify audio works
  config <msg>  Natural language configuration (e.g., "mute sound")
  automap <dir> Automatically map sound files in a directory to events using AI
  import-circus <url> <name> [event]  Import a meme sound (no slashes allowed) and optionally map to event
  analyze-log   Analyze log from stdin and play a sound
  analyze-resp  Analyze AI response from stdin and optionally override sound strategy
  skins         Download custom Winamp 2.95 skins for Claune
  geocities     Run a fake 90s-era WS_FTP terminal log to GeoCities
  website       Launch the official cyber portal in your default web browser
  help          Show this help message`)
}
