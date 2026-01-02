# Updati

Automatically update Laravel project dependencies across multiple GitHub repositories.

## Installation

### GitHub Action

Create `.github/workflows/updati.yml`:

```yaml
name: Update Dependencies

on:
  schedule:
    - cron: '0 9 * * 1' # Every Monday 9 AM
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: janyksteenbeek/updati@main
        with:
          github_token: ${{ secrets.UPDATI_TOKEN }}
          owner: your-org
          repo_patterns: |
            ^laravel-.*
            .*-api$
            my-specific-repo
```

Requires a [Personal Access Token](https://github.com/settings/tokens) with `repo` scope stored as `UPDATI_TOKEN` secret.

### Docker

```bash
docker run --rm \
  -e GITHUB_TOKEN=xxx \
  -e UPDATI_OWNER=your-org \
  ghcr.io/janyksteenbeek/updati:latest
```

### Binary

```bash
go install github.com/janyksteenbeek/updati/cmd/updati@latest

updati -t $GITHUB_TOKEN -o your-org -p "^laravel-.*"
```

## Usage

```bash
# Update all Laravel repos matching pattern
updati -t $GITHUB_TOKEN -o myorg -p "^laravel-.*"

# Multiple patterns
updati -t $GITHUB_TOKEN -o myorg -p "^laravel-.*" -p ".*-api$"

# Push directly instead of creating PRs
updati -t $GITHUB_TOKEN -o myorg --push

# Dry run
updati -t $GITHUB_TOKEN -o myorg --dry-run

# Use config file
updati -c .updati.yml
```

## Options

| Flag | Description |
|------|-------------|
| `-t, --token` | GitHub token |
| `-o, --owner` | GitHub user or org |
| `-p, --pattern` | Regex to match repos (repeatable) |
| `-w, --workers` | Concurrent workers (default: 5) |
| `-b, --base-branch` | Target branch (default: main) |
| `--push` | Push directly, no PR |
| `-n, --dry-run` | Don't make changes |
| `-c, --config` | Config file path |

## Config File

```yaml
owner: your-org
repo_patterns:
  - "^laravel-.*"
workers: 5
create_pr: true
```

## GitHub Token

Create a [Personal Access Token](https://github.com/settings/tokens) with `repo` scope.

## License

MIT
