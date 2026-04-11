package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/everlier/claune/internal/audio"
	"github.com/everlier/claune/internal/circus"
	"github.com/everlier/claune/internal/config"
)

type SoundPack struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Sounds      map[string]string `json:"sounds"`
}

var AvailablePacks = []SoundPack{
	{
		Name:        "mario",
		Description: "Classic Super Mario Bros sound effects",
		Sounds: map[string]string{
			"cli:start":    "mario-coin",
			"tool:start":   "mario-coin",
			"tool:success": "mario-1-up",
			"cli:done":     "mario-1-up",
			"tool:error":   "super-mario-death-sound-sound-effect",
			"test:fail":    "super-mario-death-sound-sound-effect",
		},
	},
	{
		Name:        "metal-gear",
		Description: "Metal Gear Solid codec and alert sounds",
		Sounds: map[string]string{
			"cli:start":    "mgs-codec",
			"tool:start":   "mgs-codec",
			"tool:error":   "metal-gear-solid-alert",
			"test:fail":    "metal-gear-solid-alert",
			"tool:success": "mgs-codec", // fallback
			"cli:done":     "mgs-codec",
		},
	},
	{
		Name:        "anime",
		Description: "Anime sound effects",
		Sounds: map[string]string{
			"cli:start":    "anime-wow",
			"tool:start":   "anime-wow",
			"tool:error":   "tuturu_1",
			"test:fail":    "tuturu_1",
			"tool:success": "tuturu_1",
			"cli:done":     "tuturu_1",
		},
	},
	{
		Name:        "retro-games",
		Description: "Retro arcade and console sounds",
		Sounds: map[string]string{
			"cli:start":    "pacman-beginning",
			"tool:start":   "sonic-ring",
			"tool:success": "zelda-secret",
			"cli:done":     "final-fantasy-victory",
			"tool:error":   "pacman-dies",
			"test:fail":    "pacman-dies",
		},
	},
	{
		Name:        "windows",
		Description: "Classic Windows operating system sounds",
		Sounds: map[string]string{
			"cli:start":    "windows-xp-startup",
			"tool:start":   "windows-95-startup",
			"tool:success": "windows-xp-hardware-insert",
			"cli:done":     "windows-xp-shutdown",
			"tool:error":   "windows-xp-error",
			"test:fail":    "windows-xp-error",
		},
	},
	{
		Name:        "half-life",
		Description: "Half-Life HEV suit and game sounds",
		Sounds: map[string]string{
			"cli:start":    "hl1_startup",
			"tool:start":   "hl1_health_pickup",
			"tool:success": "hl1_ammo_pickup",
			"cli:done":     "hl1_battery_pickup",
			"tool:error":   "hl1_flatline",
			"test:fail":    "hl1_flatline",
		},
	},
}

func handlePack(args []string) error {
	var packName string

	if len(args) < 3 {
		var options []string
		for _, pack := range AvailablePacks {
			options = append(options, fmt.Sprintf("%s - %s", pack.Name, pack.Description))
		}

		prompt := &survey.Select{
			Message: "Choose a sound pack to install:",
			Options: options,
		}

		var selectedOption string
		err := survey.AskOne(prompt, &selectedOption)
		if err != nil {
			fmt.Println("Selection canceled.")
			return nil
		}

		for _, pack := range AvailablePacks {
			if fmt.Sprintf("%s - %s", pack.Name, pack.Description) == selectedOption {
				packName = pack.Name
				break
			}
		}
	} else {
		packName = args[2]
	}

	var selectedPack *SoundPack
	if len(packName) >= 7 && (packName[:7] == "http://" || (len(packName) >= 8 && packName[:8] == "https://")) {
		importSpinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		importSpinner.Prefix = " "
		importSpinner.Suffix = fmt.Sprintf(" Fetching custom pack from %s... ", Style(packName, ColorCyan))
		importSpinner.Start()
		
		resp, err := http.Get(packName)
		importSpinner.Stop()
		if err != nil {
			return fmt.Errorf("failed to fetch custom pack: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch custom pack: HTTP %d", resp.StatusCode)
		}

		var customPack SoundPack
		if err := json.NewDecoder(resp.Body).Decode(&customPack); err != nil {
			return fmt.Errorf("failed to parse custom pack JSON from URL: %w", err)
		}
		selectedPack = &customPack
	} else if fileInfo, err := os.Stat(packName); err == nil && !fileInfo.IsDir() && filepath.Ext(packName) == ".json" {
		importSpinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		importSpinner.Prefix = " "
		importSpinner.Suffix = fmt.Sprintf(" Reading custom pack from %s... ", Style(packName, ColorCyan))
		importSpinner.Start()

		file, err := os.Open(packName)
		importSpinner.Stop()
		if err != nil {
			return fmt.Errorf("failed to read custom pack file: %w", err)
		}
		defer file.Close()

		var customPack SoundPack
		if err := json.NewDecoder(file).Decode(&customPack); err != nil {
			return fmt.Errorf("failed to parse custom pack JSON from file: %w\nEnsure it matches the correct format: {\"name\": \"...\", \"description\": \"...\", \"sounds\": {\"event\": \"slug\"}}", err)
		}
		selectedPack = &customPack
	} else {
		for _, p := range AvailablePacks {
			if p.Name == packName {
				selectedPack = &p
				break
			}
		}
	}

	if selectedPack == nil {
		return fmt.Errorf("unknown sound pack '%s'", packName)
	}

	fmt.Printf("Installing %s sound pack...\n", Style(selectedPack.Name, ColorCyan))

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Sounds == nil {
		cfg.Sounds = make(map[string]config.EventSoundConfig)
	}

	cacheDir := audio.SoundCacheDir()

	for event, slug := range selectedPack.Sounds {
		url := fmt.Sprintf("https://www.myinstants.com/media/sounds/%s.mp3", slug)
		fileName := slug + ".mp3"

		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Prefix = " "
		s.Suffix = fmt.Sprintf(" Downloading %s... ", Style(slug, ColorCyan))
		s.Start()

		err := circus.ImportMemeSound(url, fileName)

		s.Stop()
		if err != nil {
			fmt.Printf(" ❌ %s: %s\n", Style(slug, ColorRed), err.Error())
			continue
		}
		fmt.Printf(" ✅ %s\n", Style(slug, ColorGreen))

		dest := filepath.Join(cacheDir, fileName)
		eventCfg := cfg.Sounds[event]
		eventCfg.Paths = []string{dest} // Replace existing mapping
		cfg.Sounds[event] = eventCfg
	}

	err = config.Save(cfg)
	if err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println(Style("\nSound pack installed successfully!", ColorGreen+ColorBold))
	return nil
}
