package cli

import (
	"fmt"
	"os"
)

const bashCompletion = `
_claune_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        opts="install uninstall init status version doctor completion update help play config automap import-circus analyze-log analyze-resp auth skins geocities hack website"
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi

    case "${prev}" in
        play)
            local events="cli:start tool:start tool:success tool:error cli:done tool:destructive tool:readonly build:success build:fail test:fail panic warn"
            COMPREPLY=( $(compgen -W "${events}" -- ${cur}) )
            return 0
            ;;
        automap)
            compopt -o dirnames 2>/dev/null || true
            COMPREPLY=( $(compgen -d -- ${cur}) )
            return 0
            ;;
        *)
            ;;
    esac

    return 0
}

complete -F _claune_completions claune
`

const zshCompletion = `
#compdef claune

_claune() {
    local -a commands
    commands=(
        'install:Install sound hooks into Claude Code settings'
        'uninstall:Remove sound hooks from Claude Code settings'
        'init:Create a default configuration file'
        'status:Show whether hooks are installed'
        'version:Show claune version'
        'doctor:Show system diagnostics and configuration info'
        'completion:Generate shell completion scripts'
        'update:Update claune to the latest version'
        'help:Show help message'
        'play:Play a sound for an event'
        'config:Natural language configuration'
        'automap:Automatically map sound files in a directory to events'
        'import-circus:Import a meme sound'
        'analyze-log:Analyze log from stdin and play a sound'
        'analyze-resp:Analyze AI response from stdin'
        'auth:Save API key and enable AI features'
        'skins:Download custom Winamp 2.95 skins'
        'geocities:Run a fake 90s-era WS_FTP terminal log to GeoCities'
        'hack:Hack the mainframe'
        'website:Launch the official cyber portal'
    )

    if (( CURRENT == 2 )); then
        _describe -t commands 'claune commands' commands
    elif (( CURRENT == 3 )); then
        case ${words[2]} in
            play)
                local -a events
                events=(cli:start tool:start tool:success tool:error cli:done tool:destructive tool:readonly build:success build:fail test:fail panic warn)
                _describe -t events 'events' events
                ;;
            automap)
                _path_files -/
                ;;
        esac
    fi
}

_claune
`

func runCompletion(shell string) {
	switch shell {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s. Supported shells are bash, zsh\n", shell)
		os.Exit(1)
	}
}
