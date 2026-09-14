package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/chorus/messenger/internal/config"
	"github.com/chorus/messenger/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Overload()
	cfg := config.Load()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	runner := database.NewSeedRunner(db)
	ctx := context.Background()

	switch command {
	case "check":
		checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
		pair := checkCmd.String("pair", "en-es", "Language pair (e.g. en-es)")
		_ = checkCmd.Parse(os.Args[2:])

		results, err := runner.Check(ctx, *pair)
		if err != nil {
			log.Fatalf("Check failed: %v", err)
		}
		fmt.Printf("Seed check results for pair %q:\n", *pair)
		for file, changed := range results {
			if changed {
				fmt.Printf("  [PENDING] %s\n", file)
			} else {
				fmt.Printf("  [UP-TO-DATE] %s\n", file)
			}
		}

	case "apply":
		applyCmd := flag.NewFlagSet("apply", flag.ExitOnError)
		pair := applyCmd.String("pair", "", "Language pair (e.g. en-es)")
		all := applyCmd.Bool("all", false, "Apply seeds for all language pairs")
		_ = applyCmd.Parse(os.Args[2:])

		if *all {
			applied, err := runner.RunAll(ctx)
			if err != nil {
				log.Fatalf("Apply all failed: %v", err)
			}
			fmt.Printf("Applied %d seed files across all pairs.\n", len(applied))
		} else if *pair != "" {
			applied, err := runner.Run(ctx, *pair)
			if err != nil {
				log.Fatalf("Apply pair %q failed: %v", *pair, err)
			}
			fmt.Printf("Applied %d seed files for pair %q.\n", len(applied), *pair)
		} else {
			fmt.Println("Please specify either --pair <pair> or --all")
			applyCmd.Usage()
			os.Exit(1)
		}

	case "status":
		status, err := runner.Status(ctx)
		if err != nil {
			log.Fatalf("Status failed: %v", err)
		}
		fmt.Println("Applied Seed Checksums in Database:")
		for file, checksum := range status {
			fmt.Printf("  %-45s %s\n", file, checksum[:12]+"...")
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Chorus Seed CLI")
	fmt.Println("Usage:")
	fmt.Println("  go run ./cmd/seed check --pair en-es      # dry run check")
	fmt.Println("  go run ./cmd/seed apply --pair en-es      # apply single pair")
	fmt.Println("  go run ./cmd/seed apply --all             # apply all pairs")
	fmt.Println("  go run ./cmd/seed status                  # list last applied checksums")
}
