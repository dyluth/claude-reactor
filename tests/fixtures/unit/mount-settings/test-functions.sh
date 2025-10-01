update_claude_settings_for_mounts() {
    # Only update settings if we have mount paths
    if [ ${#MOUNT_PATHS[@]} -eq 0 ]; then
        log_verbose "No mount paths to configure"
        return
    fi
    
    local settings_file=".claude/settings.local.json"
    local claude_dir=".claude"
    
    # Create .claude directory if it doesn't exist
    mkdir -p "$claude_dir"
    
    # Build array of mount paths as they appear in container
    local mount_dirs=()
    for mount_path in "${MOUNT_PATHS[@]}"; do
        local basename=$(basename "$mount_path")
        mount_dirs+=("/mnt/$basename")
    done
    
    # Read existing settings or create empty object
    local existing_settings="{}"
    if [ -f "$settings_file" ]; then
        if command -v jq >/dev/null 2>&1 && jq empty "$settings_file" >/dev/null 2>&1; then
            existing_settings=$(cat "$settings_file")
            log_verbose "Read existing Claude settings from $settings_file"
        else
            log_verbose "Existing settings file has invalid JSON, creating new one"
        fi
    fi
    
    # Build new additionalDirectories array (preserve existing + add mounts)
    local existing_dirs=""
    if command -v jq >/dev/null 2>&1; then
        # Get existing additionalDirectories, default to empty array if not present
        existing_dirs=$(echo "$existing_settings" | jq -r '.additionalDirectories // [] | @json' 2>/dev/null || echo "[]")
        
        # Create JSON array of mount directories
        local mount_dirs_json=$(printf '%s\n' "${mount_dirs[@]}" | jq -R . | jq -s .)
        
        # Merge existing and new directories, removing duplicates
        local combined_dirs=$(echo "$existing_dirs $mount_dirs_json" | jq -s 'add | unique')
        
        # Update settings with merged directories, preserving all other settings
        local updated_settings=$(echo "$existing_settings" | jq --argjson dirs "$combined_dirs" '.additionalDirectories = $dirs')
        
        # Write updated settings
        echo "$updated_settings" | jq . > "$settings_file"
        
        local mount_count=${#mount_dirs[@]}
        log_info "📁 Updated Claude settings with $mount_count mounted director$([ $mount_count -eq 1 ] && echo "y" || echo "ies")"
        log_verbose "Mount directories: ${mount_dirs[*]}"
        log_verbose "Settings file: $settings_file"
    else
        log_warning "jq not found, cannot update Claude settings file"
        log_verbose "Install jq to automatically configure mounted directories for Claude"
    fi
}
