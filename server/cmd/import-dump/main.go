package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"foxygen-vibe/server/internal/legacyimport/agreements"
	"foxygen-vibe/server/internal/legacyimport/attachments"
	"foxygen-vibe/server/internal/legacyimport/classificators"
	"foxygen-vibe/server/internal/legacyimport/clients"
	"foxygen-vibe/server/internal/legacyimport/contacts"
	"foxygen-vibe/server/internal/legacyimport/devices"
	"foxygen-vibe/server/internal/legacyimport/manufacturers"
	"foxygen-vibe/server/internal/legacyimport/regions"
	"foxygen-vibe/server/internal/legacyimport/researchtypes"
	"foxygen-vibe/server/internal/legacyimport/ticketreasons"
	"foxygen-vibe/server/internal/legacyimport/tickets"
	"foxygen-vibe/server/internal/legacyimport/ticketstatuses"
	"foxygen-vibe/server/internal/legacyimport/tickettypes"
	"foxygen-vibe/server/internal/legacyimport/users"
)

type importStep struct {
	Name          string
	Package       string
	NeedsPassword bool
}

var importSteps = []importStep{
	{Name: "regions", Package: "./cmd/import-regions"},
	{Name: "clients", Package: "./cmd/import-clients"},
	{Name: "contacts", Package: "./cmd/import-contacts"},
	{Name: "research-types", Package: "./cmd/import-research-types"},
	{Name: "manufacturers", Package: "./cmd/import-manufacturers"},
	{Name: "classificators", Package: "./cmd/import-classificators"},
	{Name: "devices", Package: "./cmd/import-devices"},
	{Name: "ticket-statuses", Package: "./cmd/import-ticket-statuses"},
	{Name: "ticket-types", Package: "./cmd/import-ticket-types"},
	{Name: "ticket-reasons", Package: "./cmd/import-ticket-reasons"},
	{Name: "users", Package: "./cmd/import-users", NeedsPassword: true},
	{Name: "tickets", Package: "./cmd/import-tickets"},
	{Name: "attachments", Package: "./cmd/import-attachments"},
	{Name: "agreements", Package: "./cmd/import-agreements"},
}

type runOptions struct {
	SourcePath      string
	DefaultPassword string
	DryRun          bool
	Timeout         time.Duration
	PerUserTimeout  time.Duration
}

func main() {
	var (
		sourcePath      string
		defaultPassword string
		only            string
		dryRun          bool
		timeout         time.Duration
		perUserTimeout  time.Duration
		keepGoing       bool
		listOnly        bool
	)

	flag.StringVar(&sourcePath, "source", defaultSourcePath(), "Path to the legacy CouchDB _all_docs JSON export")
	flag.StringVar(&defaultPassword, "default-password", "", "Temporary password for imported users")
	flag.StringVar(&only, "only", "", "Comma-separated subset of import steps to run")
	flag.BoolVar(&dryRun, "dry-run", false, "Parse and plan each import step without writing to PostgreSQL")
	flag.DurationVar(&timeout, "timeout", 0, "Override each import command timeout")
	flag.DurationVar(&perUserTimeout, "per-user-timeout", 0, "Override the per-user timeout for the users import")
	flag.BoolVar(&keepGoing, "keep-going", false, "Continue running later steps after a failure")
	flag.BoolVar(&listOnly, "list", false, "Print the available import steps and exit")
	flag.Parse()

	if listOnly {
		for _, step := range importSteps {
			log.Println(step.Name)
		}
		return
	}

	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		log.Fatal("missing dump.json path")
	}
	if _, err := os.Stat(sourcePath); err != nil {
		log.Fatalf("dump file %q is not available: %v", sourcePath, err)
	}

	steps, err := selectSteps(only)
	if err != nil {
		log.Fatal(err)
	}
	if requiresDefaultPassword(steps) && strings.TrimSpace(defaultPassword) == "" {
		log.Fatal("missing required -default-password")
	}

	options := runOptions{
		SourcePath:      sourcePath,
		DefaultPassword: strings.TrimSpace(defaultPassword),
		DryRun:          dryRun,
		Timeout:         timeout,
		PerUserTimeout:  perUserTimeout,
	}

	failures := 0
	for _, step := range steps {
		log.Printf("running %s", step.Name)

		if err := runStep(step, options); err != nil {
			failures++
			log.Printf("step %s failed: %v", step.Name, err)
			if !keepGoing {
				log.Fatalf("import stopped after %s failed", step.Name)
			}
			continue
		}

		log.Printf("completed %s", step.Name)
	}

	if failures > 0 {
		log.Fatalf("import finished with %d failed step(s)", failures)
	}

	log.Printf("import finished successfully (%d step(s))", len(steps))
}

func defaultSourcePath() string {
	candidates := []string{
		"dump.json",
		filepath.Join("server", "dump.json"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return "dump.json"
}

func selectSteps(raw string) ([]importStep, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "all") {
		steps := make([]importStep, len(importSteps))
		copy(steps, importSteps)
		return steps, nil
	}

	selected := make([]importStep, 0)
	seen := make(map[string]bool)

	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}

		step, ok := lookupStep(name)
		if !ok {
			return nil, fmt.Errorf("unknown import step %q", name)
		}
		if seen[step.Name] {
			continue
		}

		seen[step.Name] = true
		selected = append(selected, step)
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no import steps selected")
	}

	return selected, nil
}

func lookupStep(name string) (importStep, bool) {
	for _, step := range importSteps {
		if step.Name == name {
			return step, true
		}
	}

	return importStep{}, false
}

func requiresDefaultPassword(steps []importStep) bool {
	for _, step := range steps {
		if step.NeedsPassword {
			return true
		}
	}

	return false
}

func runStep(step importStep, options runOptions) error {
	switch step.Name {
	case "regions":
		return regions.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "clients":
		return clients.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "contacts":
		return contacts.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "research-types":
		return researchtypes.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "manufacturers":
		return manufacturers.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "classificators":
		return classificators.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "devices":
		return devices.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "ticket-statuses":
		return ticketstatuses.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "ticket-types":
		return tickettypes.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "ticket-reasons":
		return ticketreasons.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "users":
		return users.Run(options.SourcePath, options.DefaultPassword, options.DryRun, options.Timeout, options.PerUserTimeout)
	case "tickets":
		return tickets.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "attachments":
		return attachments.Run(options.SourcePath, options.DryRun, options.Timeout)
	case "agreements":
		return agreements.Run(options.SourcePath, options.DryRun, options.Timeout)
	}

	return fmt.Errorf("unsupported import step %q", step.Name)
}
