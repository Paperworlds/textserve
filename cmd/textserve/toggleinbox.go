package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/paperworlds/textserve/internal/toggleinbox"
)

func newToggleInboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toggle-inbox",
		Short: "Manage pending bundle toggle requests",
		RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}
	cmd.AddCommand(
		newToggleInboxListCmd(),
		newToggleInboxApproveCmd(),
		newToggleInboxDenyCmd(),
		newToggleInboxRequestCmd(),
	)
	return cmd
}

func newToggleInboxListCmd() *cobra.Command {
	var allStatuses bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pending toggle requests (use --all for resolved too)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			status := toggleinbox.StatusPending
			if allStatuses {
				status = ""
			}
			entries, err := toggleinbox.List(status)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("no toggle requests")
				return nil
			}
			fmt.Printf("%-40s  %-10s  %-8s  %-9s  %s\n", "ID", "BUNDLE", "ACTION", "STATUS", "REQUESTER")
			fmt.Printf("%-40s  %-10s  %-8s  %-9s  %s\n", "--", "------", "------", "------", "---------")
			for _, e := range entries {
				fmt.Printf("%-40s  %-10s  %-8s  %-9s  %s\n", e.ID, e.Bundle, e.Action, e.Status, e.Requester)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&allStatuses, "all", false, "include resolved entries (approved/denied)")
	return cmd
}

func newToggleInboxApproveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve a pending toggle request and apply it to the overlay state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := toggleinbox.Approve(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("approved %s: bundle %q → %s\n", e.ID, e.Bundle, e.Action)
			return nil
		},
	}
}

func newToggleInboxDenyCmd() *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "deny <id>",
		Short: "Deny a pending toggle request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := toggleinbox.Deny(args[0], reason)
			if err != nil {
				return err
			}
			fmt.Printf("denied %s\n", e.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "human-readable reason for denial")
	return cmd
}

func newToggleInboxRequestCmd() *cobra.Command {
	var requester string
	cmd := &cobra.Command{
		Use:   "request <bundle> <enable|disable>",
		Short: "Enqueue a new toggle request (mostly used by tests; MCP server is the normal producer)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := toggleinbox.Request(args[0], args[1], requester)
			if err != nil {
				return err
			}
			fmt.Printf("queued %s (status: %s)\n", e.ID, e.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&requester, "requester", "cli", "label recorded as the request source")
	return cmd
}
