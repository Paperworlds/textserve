package main

import (
	"testing"
)

func TestUpDown_CommandsRegistered(t *testing.T) {
	root := buildRoot()
	var foundUp, foundDown bool
	for _, sub := range root.Commands() {
		switch sub.Use {
		case "up [name]":
			foundUp = true
		case "down [name]":
			foundDown = true
		}
	}
	if !foundUp {
		t.Error("up command not registered on root")
	}
	if !foundDown {
		t.Error("down command not registered on root")
	}
}

func TestUp_RequiresTarget(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"up"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no name/tag/--all given")
	}
}

func TestDown_RequiresTarget(t *testing.T) {
	root := buildRoot()
	root.SetArgs([]string{"down"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no name/tag/--all given")
	}
}
