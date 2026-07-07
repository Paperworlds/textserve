package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestBundleCmd_Registered(t *testing.T) {
	root := buildRoot()
	for _, sub := range root.Commands() {
		if sub.Use == "bundle" {
			return
		}
	}
	t.Error("bundle command not registered on root")
}

func TestBundleCmd_ProfileAlias(t *testing.T) {
	root := buildRoot()
	for _, sub := range root.Commands() {
		if sub.Use == "bundle" {
			for _, a := range sub.Aliases {
				if a == "profile" {
					return
				}
			}
			t.Error("bundle command missing `profile` alias")
			return
		}
	}
	t.Error("bundle command not found")
}

func TestBundleCmd_SubcommandsRegistered(t *testing.T) {
	root := buildRoot()
	var bundleCmd *cobra.Command
	for _, sub := range root.Commands() {
		if sub.Use == "bundle" {
			bundleCmd = sub
			break
		}
	}
	if bundleCmd == nil {
		t.Fatal("bundle command not found")
	}
	want := map[string]bool{"list": false, "show <name>": false, "use <name>": false}
	for _, sub := range bundleCmd.Commands() {
		if _, ok := want[sub.Use]; ok {
			want[sub.Use] = true
		}
	}
	for use, found := range want {
		if !found {
			t.Errorf("bundle subcommand %q not registered", use)
		}
	}
}

func TestBundleUse_RequiresArg(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"bundle", "use"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when bundle use called with no args")
	}
}

func TestBundleShow_RequiresArg(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"bundle", "show"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when bundle show called with no args")
	}
}
