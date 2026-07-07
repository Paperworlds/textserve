package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

func newModeCmd() *cobra.Command {
	var all bool
	c := &cobra.Command{
		Use:   "mode <server> [tool] readonly|readwrite",
		Short: "Set readonly/readwrite mode on a live adapter (resets on restart)",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			var serverName, tool, modeArg string
			if all || len(args) == 2 {
				serverName, modeArg = args[0], args[len(args)-1]
			} else {
				serverName, tool, modeArg = args[0], args[1], args[2]
			}

			var readonly bool
			switch modeArg {
			case "readonly":
				readonly = true
			case "readwrite":
				readonly = false
			default:
				return fmt.Errorf("mode must be 'readonly' or 'readwrite', got %q", modeArg)
			}

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

			body, _ := json.Marshal(map[string]bool{"readonly": readonly})
			client := &http.Client{Timeout: 5 * time.Second}

			var url string
			if all || tool == "" {
				url = fmt.Sprintf("http://localhost:%d/mode", cfg.Port)
			} else {
				url = fmt.Sprintf("http://localhost:%d/mode/%s", cfg.Port, tool)
			}

			resp, err := client.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("POST %s: %w", url, err)
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, raw)
			}

			var result map[string]any
			if err := json.Unmarshal(raw, &result); err != nil {
				return fmt.Errorf("unexpected response: %s", raw)
			}

			if all || tool == "" {
				tools := result["tools"]
				fmt.Fprintf(cmd.OutOrStdout(), "%s/* → %s (%v)\n", serverName, result["mode"], tools)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s/%s → %s\n", serverName, tool, result["mode"])
			}
			return nil
		},
	}
	c.Flags().BoolVar(&all, "all", false, "set mode on all adapters in the server")
	return c
}
