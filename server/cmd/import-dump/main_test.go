package main

import "testing"

func TestSelectStepsDefaultsToAll(t *testing.T) {
	t.Parallel()

	steps, err := selectSteps("")
	if err != nil {
		t.Fatalf("selectSteps: %v", err)
	}

	if len(steps) != len(importSteps) {
		t.Fatalf("expected %d steps, got %d", len(importSteps), len(steps))
	}

	if steps[0].Name != importSteps[0].Name || steps[len(steps)-1].Name != importSteps[len(importSteps)-1].Name {
		t.Fatalf("expected default order to match the configured import order")
	}
}

func TestSelectStepsKeepsDeclaredOrder(t *testing.T) {
	t.Parallel()

	steps, err := selectSteps("users,regions,users,tickets")
	if err != nil {
		t.Fatalf("selectSteps: %v", err)
	}

	if len(steps) != 3 {
		t.Fatalf("expected 3 unique steps, got %d", len(steps))
	}

	if steps[0].Name != "users" || steps[1].Name != "regions" || steps[2].Name != "tickets" {
		t.Fatalf("unexpected step order: %#v", steps)
	}
}

func TestRequiresDefaultPassword(t *testing.T) {
	t.Parallel()

	needsPassword := requiresDefaultPassword([]importStep{
		{
			Name:          "users",
			Package:       "./cmd/import-users",
			NeedsPassword: true,
		},
	})

	if !needsPassword {
		t.Fatal("expected users step to require a default password")
	}
}

func TestRequiresDefaultPasswordIgnoresOtherSteps(t *testing.T) {
	t.Parallel()

	if requiresDefaultPassword([]importStep{{Name: "regions", Package: "./cmd/import-regions"}}) {
		t.Fatal("expected non-user steps to skip default password validation")
	}
}
