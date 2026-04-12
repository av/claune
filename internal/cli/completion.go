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
        opts="install uninstall init setup status version doctor completion update logs help play config automap import-circus pack analyze-log analyze-resp auth mute unmute notify volume"
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi

    case "${prev}" in
        pack)
            local packs="mario metal-gear anime"
            COMPREPLY=( $(compgen -W "${packs}" -- ${cur}) )
            return 0
            ;;
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
        'setup:Run the interactive first-run wizard'
        'status:Show whether hooks are installed'
        'version:Show claune version'
        'doctor:Show system diagnostics and configuration info'
        'completion:Generate shell completion scripts'
                'update:Update claune to the latest version'
        'mute:Mute all sound effects'
        'unmute:Unmute all sound effects'
        'volume:Set the global volume level'
        'logs:View or clear application logs'
        'help:Show help message'
        'play:Play a sound for an event'
        'config:Natural language configuration'
        'automap:Automatically map sound files in a directory to events'
        'import-circus:Import a meme sound'
        'pack:Download and install a pre-configured sound pack'
        'analyze-log:Analyze log from stdin and play a sound'
        'analyze-resp:Analyze AI response from stdin'
        'auth:Save API key and enable AI features'
    )

    if (( CURRENT == 2 )); then
        _describe -t commands 'claune commands' commands
    elif (( CURRENT == 3 )); then
        case ${words[2]} in
            pack)
                local -a packs
                packs=(mario metal-gear anime)
                _describe -t packs 'packs' packs
                ;;
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

const powershellCompletion = `
using namespace System.Management.Automation
using namespace System.Management.Automation.Language

Register-ArgumentCompleter -Native -CommandName 'claune' -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $commands = @(
        'install', 'uninstall', 'init', 'setup', 'status', 'version', 'doctor',
        'completion', 'update', 'logs', 'help', 'play', 'config', 'automap',
        'import-circus', 'pack', 'analyze-log', 'analyze-resp', 'auth',
        'skins', 'geocities', 'hack', 'website', 'mute', 'unmute', 'notify', 'volume'
    )

    $events = @(
        'cli:start', 'tool:start', 'tool:success', 'tool:error', 'cli:done',
        'tool:destructive', 'tool:readonly', 'build:success', 'build:fail',
        'test:fail', 'panic', 'warn'
    )
    
    $packs = @('mario', 'metal-gear', 'anime')

    $commandElements = $commandAst.CommandElements
    $commandLength = $commandElements.Count

    if ($commandLength -eq 1 -or ($commandLength -eq 2 -and $wordToComplete -ne '')) {
        $commands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
    }
    elseif ($commandLength -ge 2) {
        $prevWord = $commandElements[$commandLength - 2].Extent.Text
        if ($wordToComplete -ne '') {
            $prevWord = $commandElements[$commandLength - 3].Extent.Text
        }

        if ($prevWord -eq 'play') {
            $events | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
        }
        elseif ($prevWord -eq 'pack') {
            $packs | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
        }
        elseif ($prevWord -eq 'automap') {
            # Let PowerShell handle directory completion
        }
    }
}
`

func runCompletion(shell string) {
	switch shell {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "powershell":
		fmt.Print(powershellCompletion)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s. Supported shells are bash, zsh, powershell\n", shell)
		os.Exit(1)
	}
}
