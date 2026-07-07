package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestProfileCmd_Registered(t *testing.T) {
	root := buildRoot()
	for _, sub := range root.Commands() {
		if sub.Use == "profile" {
			return
		}
	}
	t.Error("profile command not registered on root")
}

func TestProfileCmd_SubcommandsRegistered(t *testing.T) {
	root := buildRoot()
	var profileCmd *cobra.Command
	for _, sub := range root.Commands() {
		if sub.Use == "profile" {
			profileCmd = sub
			break
		}
	}
	if profileCmd == nil {
		t.Fatal("profile command not found")
	}
	want := map[string]bool{"list": false, "show <name>": false, "use <name>": false}
	for _, sub := range profileCmd.Commands() {
		if _, ok := want[sub.Use]; ok {
			want[sub.Use] = true
		}
	}
	for use, found := range want {
		if !found {
			t.Errorf("profile subcommand %q not registered", use)
		}
	}
}

func TestProfileUse_RequiresArg(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"profile", "use"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when profile use called with no args")
	}
}

func TestProfileShow_RequiresArg(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"profile", "show"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when profile show called with no args")
	}
}
