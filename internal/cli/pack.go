package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/everlier/claune/internal/audio"
	"github.com/everlier/claune/internal/circus"
	"github.com/everlier/claune/internal/config"
)

type SoundPack struct {
	Name        string
	Description string
	Sounds      map[string]string // Event -> Slug
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
}

func handlePack() {
	if len(os.Args) < 3 {
		fmt.Println(Style("Available Sound Packs:", ColorCyan+ColorBold))
		for _, pack := range AvailablePacks {
			fmt.Printf("  %s - %s\n", Style(fmt.Sprintf("%-12s", pack.Name), ColorGreen), pack.Description)
		}
		fmt.Println("\nUsage: claune pack <name>")
		os.Exit(1)
	}

	packName := os.Args[2]
	var selectedPack *SoundPack
	for _, p := range AvailablePacks {
		if p.Name == packName {
			selectedPack = &p
			break
		}
	}

	if selectedPack == nil {
		PrintError("Unknown sound pack '%s'", packName)
		os.Exit(1)
	}

	fmt.Printf("Installing %s sound pack...\n", Style(selectedPack.Name, ColorCyan))

	cfg, err := config.Load()
	if err != nil {
		PrintError("loading config: %v", err)
		os.Exit(1)
	}

	if cfg.Sounds == nil {
		cfg.Sounds = make(map[string]config.EventSoundConfig)
	}

	cacheDir := audio.SoundCacheDir()

	for event, slug := range selectedPack.Sounds {
		url := fmt.Sprintf("https://www.myinstants.com/media/sounds/%s.mp3", slug)
		fileName := slug + ".mp3"
		
		fmt.Printf("Downloading %s... ", slug)
		err := circus.ImportMemeSound(url, fileName)
		if err != nil {
			fmt.Println(Style("Failed: ", ColorRed) + err.Error())
			continue
		}
		fmt.Println(Style("Done", ColorGreen))

		dest := filepath.Join(cacheDir, fileName)
		eventCfg := cfg.Sounds[event]
		eventCfg.Paths = []string{dest} // Replace existing mapping
		cfg.Sounds[event] = eventCfg
	}

	err = config.Save(cfg)
	if err != nil {
		PrintError("saving config: %v", err)
		os.Exit(1)
	}

	fmt.Println(Style("\nSound pack installed successfully!", ColorGreen+ColorBold))
}
