package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/paperworlds/textserve/internal/registry"
)

// addTemplateData holds values interpolated into the server scaffold templates.
type addTemplateData struct {
	Name     string
	Image    string
	Protocol string
	Runtime  string
	Port     int
	TagsCSV  string
}

const serverYAMLTmpl = `image: "{{.Image}}"
protocol: {{.Protocol}}
runtime: {{.Runtime}}
port: {{.Port}}
container_port: {{.Port}}
endpoint_path: /mcp
tags: [{{.TagsCSV}}]
env: []
volumes: []
extra_args: []
deps: []
health:
  endpoint: /health
  timeout: 5
`

const hookShTmpl = `#!/usr/bin/env bash
# pre-start hook for {{.Name}}
# Use for side effects only (port-forwards, precondition checks).
# Credential injection belongs in server.yaml env[] entries.
`

const readmeTmpl = `# {{.Name}}

<!-- TODO: describe what this server does and what tools it exposes -->

## Protocol / Runtime

- **Protocol:** {{.Protocol}}
- **Runtime:** {{.Runtime}}
- **Port:** {{.Port}}
- **Endpoint:** http://localhost:{{.Port}}/mcp

## Auth

<!-- TODO: specify the 1Password item and fields used for credentials -->

## Prerequisites

<!-- TODO: list any deps or preconditions (VPN, port-forwards, etc.) -->

## Usage

` + "```" + `bash
textserve start {{.Name}}
claude mcp add --transport http {{.Name}} http://localhost:{{.Port}}/mcp
` + "```" + `
`

func newAddCmd() *cobra.Command {
	var (
		transport string
		port      int
		image     string
		tagsFlag  string
	)
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Scaffold a new MCP server entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fleet, repoRoot, err := loadFleet()
			if err != nil {
				return err
			}

			// Abort if servers/<name>/ already exists.
			serverDir := filepath.Join(repoRoot, "servers", name)
			if _, err := os.Stat(serverDir); err == nil {
				return fmt.Errorf("servers/%s/ already exists — aborting", name)
			}

			// Auto-assign port if not given.
			if port == 0 {
				port, err = nextAvailablePort(fleet, 9880, 9899)
				if err != nil {
					return err
				}
			}

			// Build tags list.
			var tags []string
			if tagsFlag != "" {
				for _, t := range strings.Split(tagsFlag, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						tags = append(tags, t)
					}
				}
			}
			protocol := transport
			runtime := "docker"
			if transport != "http" {
				runtime = transport
			}
			if protocol == "http" && !containsTag(tags, "docker") {
				tags = append(tags, "docker")
			}

			data := addTemplateData{
				Name:     name,
				Image:    image,
				Protocol: protocol,
				Runtime:  runtime,
				Port:     port,
				TagsCSV:  strings.Join(tags, ", "),
			}

			// Create servers/<name>/ directory.
			if err := os.MkdirAll(serverDir, 0o755); err != nil {
				return fmt.Errorf("create server dir: %w", err)
			}

			// Write server.yaml.
			if err := writeTemplate(filepath.Join(serverDir, "server.yaml"), serverYAMLTmpl, data); err != nil {
				return fmt.Errorf("write server.yaml: %w", err)
			}

			// Write hook.sh.
			hookPath := filepath.Join(serverDir, "hook.sh")
			if err := writeTemplate(hookPath, hookShTmpl, data); err != nil {
				return fmt.Errorf("write hook.sh: %w", err)
			}
			if err := os.Chmod(hookPath, 0o755); err != nil {
				return fmt.Errorf("chmod hook.sh: %w", err)
			}

			// Write README.md.
			if err := writeTemplate(filepath.Join(serverDir, "README.md"), readmeTmpl, data); err != nil {
				return fmt.Errorf("write README.md: %w", err)
			}

			// Append to registry.yaml.
			if err := appendToRegistry(repoRoot, name, protocol, port, image, tags); err != nil {
				return fmt.Errorf("update registry.yaml: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created servers/%s/\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "  server.yaml  — edit env[], deps, health as needed\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  hook.sh      — add side-effect setup here\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  README.md    — fill in auth and tool docs\n")
			fmt.Fprintf(cmd.OutOrStdout(), "\nNext steps:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  1. Edit servers/%s/server.yaml\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "  2. textserve start %s\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "  3. claude mcp add --transport %s %s http://localhost:%d/mcp\n", protocol, name, port)
			return nil
		},
	}
	cmd.Flags().StringVar(&transport, "transport", "http", "transport type (http, native, stdio)")
	cmd.Flags().IntVar(&port, "port", 0, "host port (auto-assigned in 9880-9899 if omitted)")
	cmd.Flags().StringVar(&image, "image", "", "Docker image name")
	cmd.Flags().StringVar(&tagsFlag, "tags", "", "comma-separated tags")
	return cmd
}

func writeTemplate(path, tmplStr string, data addTemplateData) error {
	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}

// nextAvailablePort returns the lowest port in [min, max] not already used in registry.
func nextAvailablePort(fleet *registry.FleetRegistry, min, max int) (int, error) {
	used := map[int]bool{}
	for _, entry := range fleet.Servers {
		if entry.Port > 0 {
			used[entry.Port] = true
		}
	}
	for p := min; p <= max; p++ {
		if !used[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available port in %d-%d range", min, max)
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// appendToRegistry adds a new entry to registry.yaml.
func appendToRegistry(repoRoot, name, protocol string, port int, image string, tags []string) error {
	regPath := filepath.Join(repoRoot, "registry.yaml")
	data, err := os.ReadFile(regPath)
	if err != nil {
		return err
	}

	// Parse existing registry to check for duplicate.
	var fleet registry.FleetRegistry
	if err := yaml.Unmarshal(data, &fleet); err != nil {
		return err
	}
	if _, exists := fleet.Servers[name]; exists {
		return fmt.Errorf("server %q already in registry", name)
	}

	// Build the YAML block for the new entry.
	tagsYAML := "[]"
	if len(tags) > 0 {
		tagsYAML = "[" + strings.Join(tags, ", ") + "]"
	}
	portLine := ""
	if port > 0 {
		portLine = fmt.Sprintf("\n    port: %d\n    container_port: %d\n    endpoint_path: /mcp", port, port)
	}
	imageLine := ""
	if image != "" {
		imageLine = fmt.Sprintf("\n    image: \"%s\"", image)
	}
	runtime := "docker"
	if protocol != "http" {
		runtime = protocol
	}
	entry := fmt.Sprintf("\n  %s:%s\n    protocol: %s\n    runtime: %s%s\n    tags: %s\n    deps: []\n    health:\n      endpoint: /health\n      timeout: 5\n",
		name, imageLine, protocol, runtime, portLine, tagsYAML)

	return os.WriteFile(regPath, append(data, []byte(entry)...), 0o644)
}
