# Fish shell completions for textserve
# Install: cp completions/textserve.fish ~/.config/fish/completions/textserve.fish

# Disable file completions for textserve
complete -c textserve -f

# Subcommands
set -l subcommands start stop restart up down logs list status health doctor preflight add

for sub in $subcommands
    complete -c textserve -n "__fish_use_subcommand $subcommands" -a $sub
end

# Server names (dynamic: from textserve list)
set -l server_cmds start stop restart logs health
for sub in $server_cmds
    complete -c textserve -n "__fish_seen_subcommand_from $sub" \
        -a "(textserve list 2>/dev/null)" \
        -d "MCP server"
end

# --tag flag (single tag) for: start stop restart list status health
set -l tag_values ci docker data monitoring comms native stdio
set -l tag_cmds start stop restart up down list status health
for sub in $tag_cmds
    for val in $tag_values
        complete -c textserve -n "__fish_seen_subcommand_from $sub" \
            -l tag -a $val -d "filter by tag"
    end
end

# preflight --tags (comma-separated, same values)
for val in $tag_values
    complete -c textserve -n "__fish_seen_subcommand_from preflight" \
        -l tags -a $val -d "filter by tags"
end

# logs --follow / -f
complete -c textserve -n "__fish_seen_subcommand_from logs" \
    -l follow -s f -d "follow log output"

# preflight --json
complete -c textserve -n "__fish_seen_subcommand_from preflight" \
    -l json -d "emit JSON report"

# add flags
complete -c textserve -n "__fish_seen_subcommand_from add" \
    -l transport -a "http native stdio" -d "transport type"
complete -c textserve -n "__fish_seen_subcommand_from add" \
    -l port -d "host port"
complete -c textserve -n "__fish_seen_subcommand_from add" \
    -l image -d "Docker image"
complete -c textserve -n "__fish_seen_subcommand_from add" \
    -l tags -d "comma-separated tags"
