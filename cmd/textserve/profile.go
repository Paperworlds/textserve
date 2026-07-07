package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/paperworlds/textserve/internal/registry"
)

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage session profiles (named sets of MCP servers)",
		RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}
	cmd.AddCommand(newProfileListCmd(), newProfileShowCmd(), newProfileUseCmd())
	return cmd
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, _, err := loadFleet()
			if err != nil {
				return err
			}
			if len(fleet.Profiles) == 0 {
				fmt.Println("no profiles defined in registry.yaml")
				return nil
			}
			names := make([]string, 0, len(fleet.Profiles))
			for name := range fleet.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)

			fmt.Printf("%-16s  %-7s  %s\n", "PROFILE", "SERVERS", "DESCRIPTION")
			fmt.Printf("%-16s  %-7s  %s\n", "-------", "-------", "-----------")
			for _, name := range names {
				p := fleet.Profiles[name]
				resolved, err := fleet.ResolveProfile(name)
				count := 0
				if err == nil {
					count = len(resolved)
				}
				fmt.Printf("%-16s  %-7d  %s\n", name, count, p.Description)
			}
			return nil
		},
	}
}

func newProfileShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show resolved server list for a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, _, err := loadFleet()
			if err != nil {
				return err
			}
			name := args[0]
			p, ok := fleet.Profiles[name]
			if !ok {
				return fmt.Errorf("profile %q not found", name)
			}
			servers, err := fleet.ResolveProfile(name)
			if err != nil {
				return err
			}
			if p.Description != "" {
				fmt.Printf("%s — %s\n", name, p.Description)
			} else {
				fmt.Println(name)
			}
			for _, s := range servers {
				fmt.Printf("  %s\n", s)
			}
			return nil
		},
	}
}

func newProfileUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Converge the fleet to a named profile (bring up needed servers, bring down the rest)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, repoRoot, err := loadFleet()
			if err != nil {
				return err
			}
			name := args[0]
			desired, err := fleet.ResolveProfile(name)
			if err != nil {
				return err
			}

			desiredSet := map[string]bool{}
			for _, s := range desired {
				desiredSet[s] = true
			}

			all := fleet.ListNames()

			for _, n := range all {
				entry := fleet.Servers[n]
				cfg := serverConfig(repoRoot, n, entry)
				if cfg.Runtime == registry.RuntimeClaude {
					continue
				}

				if desiredSet[n] {
					// Should be up.
					resolvePreStart(repoRoot, cfg)
					action, upErr := upServer(n, cfg, repoRoot, entry)
					if upErr != nil {
						fmt.Fprintf(os.Stderr, "%s: %v\n", n, upErr)
						continue
					}
					switch action {
					case "skipped":
						fmt.Printf("%s → already running\n", n)
					case "registered":
						fmt.Printf("%s → registered\n", n)
					case "started":
						fmt.Printf("%s → started\n", n)
					}
				} else {
					// Should be down.
					if err := downServer(n, cfg); err != nil {
						fmt.Fprintf(os.Stderr, "%s: %v\n", n, err)
						continue
					}
					fmt.Printf("%s → stopped\n", n)
				}
			}
			return nil
		},
	}
}
