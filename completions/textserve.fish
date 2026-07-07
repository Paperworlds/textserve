# Fish shell completions for textserve
# Install: cp completions/textserve.fish ~/.config/fish/completions/textserve.fish

# Disable file completions for textserve
complete -c textserve -f

# Subcommands
set -l subcommands start stop restart up down logs list status health doctor preflight add profile remove

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

# profile sub-subcommands and profile name completions
complete -c textserve -n "__fish_seen_subcommand_from profile; and not __fish_seen_subcommand_from list show use" \
    -a "list" -d "list profiles"
complete -c textserve -n "__fish_seen_subcommand_from profile; and not __fish_seen_subcommand_from list show use" \
    -a "show" -d "show profile server list"
complete -c textserve -n "__fish_seen_subcommand_from profile; and not __fish_seen_subcommand_from list show use" \
    -a "use" -d "converge fleet to profile"
for sub in show use
    complete -c textserve -n "__fish_seen_subcommand_from $sub" \
        -a "(textserve profile list 2>/dev/null | tail -n +3 | awk '{print \$1}')" \
        -d "profile"
end

# remove: server names + flags
complete -c textserve -n "__fish_seen_subcommand_from remove" \
    -a "(textserve list 2>/dev/null)" -d "MCP server"
complete -c textserve -n "__fish_seen_subcommand_from remove" \
    -l global -d "remove from global config"
complete -c textserve -n "__fish_seen_subcommand_from remove" \
    -l repo -d "path to project repo"
complete -c textserve -n "__fish_seen_subcommand_from remove" \
    -l all -d "remove from all config files"
complete -c textserve -n "__fish_seen_subcommand_from remove" \
    -l dry-run -d "preview without writing"

# --profile global flag (textaccounts integration)
complete -c textserve -l profile \
    -a "(textaccounts list 2>/dev/null | awk 'NR>1 {print $1}')" \
    -d "textaccounts profile to target"

# add flags
complete -c textserve -n "__fish_seen_subcommand_from add" \
    -l transport -a "http native stdio" -d "transport type"
complete -c textserve -n "__fish_seen_subcommand_from add" \
    -l port -d "host port"
complete -c textserve -n "__fish_seen_subcommand_from add" \
    -l image -d "Docker image"
complete -c textserve -n "__fish_seen_subcommand_from add" \
    -l tags -d "comma-separated tags"
