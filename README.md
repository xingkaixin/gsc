# gsc

Google Search Console CLI in Go for checking indexing coverage, keyword performance, and SEO insights from the terminal.

Based on the MIT-licensed Go core of [piyush-gambhir/gsc-cli](https://github.com/piyush-gambhir/gsc-cli), stripped of self-update and web docs, managed with [mise](https://mise.jdx.dev/), and extended with `gsc index-audit` for automated sitemap indexing audits.

## Development & Install (mise)

Go version (`1.27.4`) and build tasks are defined in [`mise.toml`](./mise.toml):

```bash
# Local debugging via go run
mise run dev -- version
mise run dev -- sites list

# Compile binary to bin/gsc
mise run build

# Build and install into ~/.local/bin/gsc (override with INSTALL_DIR)
mise run install

# Run full test suite with race detector
mise run test
```

## Authentication

Config is stored at `~/.config/gsc/config.yaml` (override with `GSC_CONFIG` or `XDG_CONFIG_HOME`).

### Service Account (Recommended for personal use & automation)

1. Enable the [Google Search Console API](https://console.cloud.google.com/apis/library/searchconsole.googleapis.com) in Google Cloud.
2. Create a [Service Account](https://console.cloud.google.com/iam-admin/serviceaccounts) and download its JSON key to a private path outside any git repo:
   ```bash
   mkdir -p ~/.config/gsc
   mv ~/Downloads/<key>.json ~/.config/gsc/service-account.json
   chmod 600 ~/.config/gsc/service-account.json
   ```
3. Add the Service Account email under **Settings → Users and permissions** in [Google Search Console](https://search.google.com/search-console).
4. Log in with the key file:
   ```bash
   gsc auth login --service-account ~/.config/gsc/service-account.json
   gsc sites list
   gsc sites use sc-domain:example.com
   ```

For Desktop OAuth client, ADC, or impersonation setups, see [docs/auth.md](./docs/auth.md).

## Common SEO Workflows

### Indexing & Unindexed Audit

```bash
# Preview URLs discovered from sitemaps registered in GSC (zero inspection quota)
gsc index-audit --urls-only

# Audit all sitemap URLs and show only unindexed / errored pages + reason summary
gsc index-audit --only-unindexed

# Audit a specific sitemap or URL prefix, export JSON
gsc index-audit --sitemap https://example.com/sitemap.xml --prefix https://example.com/blog/ -o json

# Inspect specific URLs directly
gsc inspect https://example.com/pricing
```

> Note: Google Search Console has no public API for the full Page Indexing report. `index-audit` inspects URLs discovered from sitemaps via the URL Inspection API (2,000 inspections/day per property).

### Keywords & Search Performance

```bash
# Overall clicks, impressions, CTR, position (last 28 days vs previous period)
gsc performance --last 28d --compare previous

# Top matched queries and pages
gsc top queries --last 28d --limit 50
gsc top pages --last 28d --limit 50

# Custom query + page breakdown exported as CSV
gsc query -d query -d page --last 28d --limit 5000 -o csv > seo.csv

# Computed SEO opportunities
gsc insights striking-distance --last 28d
gsc insights low-ctr --last 28d
gsc insights cannibalization --last 3m
gsc insights decliners --last 28d --compare previous
gsc insights new-queries --last 28d
gsc insights lost-queries --last 28d
```

## Agent Skill

This repository includes an Agent Skill at [`skills/gsc`](./skills/gsc/SKILL.md) so coding agents know how to safely run `gsc` for indexing audits and SEO analysis:

```bash
# Install from GitHub
skills add xingkaixin/gsc

# Or install from local checkout
skills add .
```

## Documentation & Attribution

- [API Coverage](./docs/api-coverage.md)
- [Authentication](./docs/auth.md)
- [Compatibility](./docs/compatibility.md)
- License: [MIT](./LICENSE) — original CLI core Copyright (c) 2026 Piyush Gambhir.
