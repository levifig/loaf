# Plugin Versioning Checklist

When making changes to plugins in this repo, always update versions in **both** locations:

1. **Individual plugin.json files**: `plugins/<plugin-name>/.claude-plugin/plugin.json`
2. **Marketplace manifest**: `.claude-plugin/marketplace.json` (contains version for each plugin AND a top-level metadata version)

## All version locations to update:
- `.claude-plugin/marketplace.json` → `metadata.version`
- `.claude-plugin/marketplace.json` → each plugin entry's `version`
- `plugins/*/.claude-plugin/plugin.json` → `version`

## Quick command to bump all to X.Y.Z:
```bash
# Update marketplace
sed -i '' 's/"version": "OLD"/"version": "NEW"/g' .claude-plugin/marketplace.json

# Update all plugins
for plugin in design foundations infrastructure orchestration python rails typescript; do
  sed -i '' 's/"version": "OLD"/"version": "NEW"/' "plugins/$plugin/.claude-plugin/plugin.json"
done
```
