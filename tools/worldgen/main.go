// Command worldgen asks the configured model for a world from the command
// line, the way the menu does, and prints how it went: a quick way to try
// a description or a model without the screens.
//
//	go run ./tools/worldgen imagination "wikingowie na zamarzniętym morzu"
//	go run ./tools/worldgen investigation "gra detektywistyczna w stylu Cowboy Bebop"
//
// The settings come from the environment and .env.local, as in the game;
// NEURO_WORLD_DUMP=plik.json keeps the model's raw answer.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/config"
	"github.com/sebastianrozmus/neuro-processor/internal/llm"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "użycie: worldgen <imagination|dnd|adventure|investigation> <opis świata> [plik.json]")
		os.Exit(2)
	}

	cfg := config.Load()
	if cfg.Offline() {
		fmt.Fprintln(os.Stderr, "brak modelu: ustaw NEURO_LLM_MODEL (np. w .env.local)")
		os.Exit(1)
	}

	client := &llm.Client{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model, Context: cfg.Context}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	start := time.Now()
	spec, err := client.GenerateWorld(ctx, world.Mode(os.Args[1]), os.Args[2])
	fmt.Println("czas:", time.Since(start).Round(time.Second))

	if err != nil {
		fmt.Println("błąd:", err)
		os.Exit(1)
	}

	fmt.Printf("świat: %s (%s)\n", spec.Name, spec.Mode)
	if spec.Style != nil {
		fmt.Println("styl:", strings.ReplaceAll(spec.Style.Card(), "\n", " | "))
		fmt.Println("obraz:", spec.Style.Art)
	}

	if spec.Mode.Rooms() {
		fmt.Printf("miejsca: %d, rzeczy: %d, osoby: %d, kroki: %d, zdarzenia: %d\n", len(spec.Rooms), len(spec.Things), len(spec.People), len(spec.Steps), len(spec.Events))
		if spec.Case != nil {
			fmt.Printf("sprawa: zleca %s, sprawca %s, dowody %v\n", spec.Case.Client, spec.Case.Answer, spec.Case.Proof)
		}
	} else {
		fmt.Printf("miejsca: %d, potwory: %d, przedmioty: %d, sklepy: %d, towarzysze: %d\n", len(spec.Places), len(spec.Monsters), len(spec.Items), len(spec.Shops), len(spec.Companions))
	}

	if len(os.Args) > 3 {
		data, _ := json.MarshalIndent(spec, "", "  ")
		if err := os.WriteFile(os.Args[3], data, 0o644); err != nil {
			fmt.Println("zapis:", err)
			os.Exit(1)
		}

		fmt.Println("zapisano:", os.Args[3])
	}
}
