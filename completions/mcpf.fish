# Fish shell completions for mcpf
# Install: cp completions/mcpf.fish ~/.config/fish/completions/mcpf.fish

# Disable file completions for mcpf
complete -c mcpf -f

# Subcommands
set -l subcommands start stop restart logs list status health doctor preflight add

for sub in $subcommands
    complete -c mcpf -n "__fish_use_subcommand $subcommands" -a $sub
end

# Server names (dynamic: from mcpf list)
set -l server_cmds start stop restart logs health
for sub in $server_cmds
    complete -c mcpf -n "__fish_seen_subcommand_from $sub" \
        -a "(mcpf list 2>/dev/null)" \
        -d "MCP server"
end

# --tag flag (single tag) for: start stop restart list status health
set -l tag_values ci docker data monitoring comms native stdio
set -l tag_cmds start stop restart list status health
for sub in $tag_cmds
    for val in $tag_values
        complete -c mcpf -n "__fish_seen_subcommand_from $sub" \
            -l tag -a $val -d "filter by tag"
    end
end

# preflight --tags (comma-separated, same values)
for val in $tag_values
    complete -c mcpf -n "__fish_seen_subcommand_from preflight" \
        -l tags -a $val -d "filter by tags"
end

# logs --follow / -f
complete -c mcpf -n "__fish_seen_subcommand_from logs" \
    -l follow -s f -d "follow log output"

# preflight --json
complete -c mcpf -n "__fish_seen_subcommand_from preflight" \
    -l json -d "emit JSON report"

# add flags
complete -c mcpf -n "__fish_seen_subcommand_from add" \
    -l transport -a "http native stdio" -d "transport type"
complete -c mcpf -n "__fish_seen_subcommand_from add" \
    -l port -d "host port"
complete -c mcpf -n "__fish_seen_subcommand_from add" \
    -l image -d "Docker image"
complete -c mcpf -n "__fish_seen_subcommand_from add" \
    -l tags -d "comma-separated tags"
