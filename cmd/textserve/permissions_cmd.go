package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type adapterPermissions struct {
	Readonly bool     `json:"readonly"`
	Actions  []string `json:"actions"`
}

func newPermissionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "permissions <server>",
		Short: "Show current readonly/readwrite mode for each adapter in a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]

			fleet, repoRoot, err := loadFleet()
			if err != nil {
				return err
			}
			entry, ok := fleet.Servers[serverName]
			if !ok {
				return fmt.Errorf("unknown server %q", serverName)
			}
			cfg := serverConfig(repoRoot, serverName, entry)
			if cfg.Port == 0 {
				return fmt.Errorf("server %q has no port configured", serverName)
			}

			client := &http.Client{Timeout: 5 * time.Second}
			url := fmt.Sprintf("http://localhost:%d/permissions", cfg.Port)
			resp, err := client.Get(url)
			if err != nil {
				return fmt.Errorf("GET %s: %w", url, err)
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, raw)
			}

			var perms map[string]adapterPermissions
			if err := json.Unmarshal(raw, &perms); err != nil {
				return fmt.Errorf("unexpected response: %s", raw)
			}

			tools := make([]string, 0, len(perms))
			for t := range perms {
				tools = append(tools, t)
			}
			sort.Strings(tools)

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "%-16s %-10s  %s\n", "TOOL", "MODE", "ACTIONS")
			fmt.Fprintf(w, "%-16s %-10s  %s\n", "────", "────", "───────")
			for _, t := range tools {
				p := perms[t]
				mode := "readwrite"
				if p.Readonly {
					mode = "readonly"
				}
				actions := strings.Join(p.Actions, ", ")
				fmt.Fprintf(w, "%-16s %-10s  %s\n", t, mode, actions)
			}
			return nil
		},
	}
}
