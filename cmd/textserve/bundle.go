package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/paperworlds/textserve/internal/registry"
	"github.com/paperworlds/textserve/internal/togglestate"
)

func newBundleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundle",
		Aliases: []string{"profile"},
		Short:   "Manage togglable server bundles (formerly profiles)",
		RunE:    func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}
	cmd.AddCommand(newBundleListCmd(), newBundleShowCmd(), newBundleUseCmd())
	return cmd
}

func newBundleListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available bundles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, _, err := loadFleet()
			if err != nil {
				return err
			}
			if len(fleet.Bundles) == 0 {
				fmt.Println("no bundles defined in registry.yaml")
				return nil
			}
			names := make([]string, 0, len(fleet.Bundles))
			for name := range fleet.Bundles {
				names = append(names, name)
			}
			sort.Strings(names)

			overlay, err := togglestate.Load()
			if err != nil {
				return err
			}

			fmt.Printf("%-16s  %-7s  %-9s  %s\n", "BUNDLE", "SERVERS", "ENABLED", "DESCRIPTION")
			fmt.Printf("%-16s  %-7s  %-9s  %s\n", "------", "-------", "-------", "-----------")
			for _, name := range names {
				b := fleet.Bundles[name]
				resolved, err := fleet.ResolveBundle(name)
				count := 0
				if err == nil {
					count = len(resolved)
				}
				effective := overlay.Effective(name, b.Enabled)
				enabled := "no"
				if effective {
					enabled = "yes"
				}
				if _, ok := overlay.Lookup(name); ok && effective != b.Enabled {
					enabled += "*"
				}
				fmt.Printf("%-16s  %-7d  %-9s  %s\n", name, count, enabled, b.Description)
			}
			return nil
		},
	}
}

func newBundleShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show resolved server list for a bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, _, err := loadFleet()
			if err != nil {
				return err
			}
			name := args[0]
			b, ok := fleet.Bundles[name]
			if !ok {
				return fmt.Errorf("bundle %q not found", name)
			}
			servers, err := fleet.ResolveBundle(name)
			if err != nil {
				return err
			}
			overlay, err := togglestate.Load()
			if err != nil {
				return err
			}
			effective := overlay.Effective(name, b.Enabled)
			source := "registry"
			if _, ok := overlay.Lookup(name); ok {
				source = "overlay"
			}

			header := name
			if b.Description != "" {
				header = fmt.Sprintf("%s — %s", name, b.Description)
			}
			state := "disabled"
			if effective {
				state = "enabled"
			}
			header += fmt.Sprintf("  [%s via %s]", state, source)
			fmt.Println(header)
			for _, s := range servers {
				fmt.Printf("  %s\n", s)
			}
			return nil
		},
	}
}

func newBundleUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Converge the fleet to a named bundle (bring up needed servers, bring down the rest)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, repoRoot, err := loadFleet()
			if err != nil {
				return err
			}
			name := args[0]
			desired, err := fleet.ResolveBundle(name)
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
