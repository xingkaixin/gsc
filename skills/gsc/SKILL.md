---
name: gsc
description: >-
  Use the `gsc` CLI to inspect Google Search Console data: site indexing and
  unindexed coverage audits (`index-audit`, `inspect`), search performance
  totals, matched keywords/queries, top pages, trends, sitemaps, and SEO
  opportunity insights (`striking-distance`, `low-ctr`, `cannibalization`,
  `decliners`, `new-queries`, `lost-queries`). Activate whenever the user asks
  about Google Search Console, website indexing status, unindexed pages, SEO
  keywords, clicks/impressions/CTR/position, or sitemaps.
---

# Using the `gsc` CLI

`gsc` is a Go CLI for Google Search Console. Data is written to `stdout` and diagnostics/summaries to `stderr`.

## Agent Operating Rules

1. **Confirm authentication and target site first.**
   - Run `gsc status -o json` (or `gsc doctor --online -o json`) if unsure whether credentials or a default site are configured.
   - If the account has multiple properties, run `gsc sites list -o json` and pass `-s <site>` explicitly on every query (`-s example.com` or `-s sc-domain:example.com`). Do not assume the default site matches what the user means.
2. **Default to safe, machine-readable invocations.**
   - Prefer `--read-only --no-input -o json` for read queries so output includes `site`, date ranges, and structured fields.
   - Never run mutating commands (`sites add/remove`, `sitemaps submit/delete`, `auth logout`) without explicit user confirmation; preview remote writes with `--dry-run` first.
   - Never run `gsc auth token` or read/print credential JSON files (`service-account.json`, `secrets.yaml`).
3. **Protect URL Inspection quota (2,000/day per property).**
   - Before running `gsc index-audit`, run `gsc index-audit --urls-only -o json` first to check how many URLs are discovered from sitemaps.
   - Narrow large sites with `--prefix`, `--match`, or `--max`.
4. **Respect Search Analytics math and API boundaries.**
   - Never sum rows from `top queries` or `query` to compute site totals—Google withholds anonymized queries. Always use `gsc performance` for totals.
   - Google has no public API for the full Page Indexing report (pages outside sitemaps are not discoverable via API), and `inspect` returns the last indexed snapshot—not a live test or "Request indexing".
   - Search Analytics dates are in Pacific Time and final data typically lags 2–3 days (`gsc freshness` checks the latest available date).

## Core SEO Workflows

### 1. Check Setup & Properties

```bash
gsc status -o json
gsc sites list --read-only --no-input -o json
gsc freshness -s <site> --read-only --no-input -o json
```

### 2. Indexing & Unindexed Coverage Audit

```bash
# Step 1: Preview sitemap URL count (0 inspection quota)
gsc index-audit -s <site> --urls-only --read-only --no-input -o json

# Step 2: Audit indexing and return unindexed/errored URLs + grouped reasons
gsc index-audit -s <site> --only-unindexed --read-only --no-input -o json

# Filter to a section or explicit sitemap
gsc index-audit -s <site> --sitemap https://example.com/sitemap.xml --prefix https://example.com/blog/ --only-unindexed -o json

# Inspect specific URLs directly
gsc inspect -s <site> https://example.com/pricing https://example.com/blog/post --read-only --no-input -o json
```

When reporting `index-audit` results to the user, lead with:
- Total discovered vs. inspected URLs, indexed count & index rate (`indexed / inspected`).
- Top `unindexed_reasons` grouped by `coverage_state` (e.g., `Discovered - currently not indexed`, `Crawled - currently not indexed`).
- Concrete sample URLs for the main unindexed causes.

### 3. Overall Performance & Trends

```bash
# Site totals for last 28 days compared to previous 28 days
gsc performance -s <site> --last 28d --compare previous --read-only --no-input -o json

# Year-over-year comparison
gsc performance -s <site> --last 28d --compare yoy --read-only --no-input -o json

# Daily / weekly / monthly trend
gsc trend -s <site> --last 3m --by week --read-only --no-input -o json
```

### 4. Matched Keywords & Top Pages

```bash
# Top matched keywords (queries)
gsc top queries -s <site> --last 28d --limit 50 --compare previous --read-only --no-input -o json

# Top landing pages
gsc top pages -s <site> --last 28d --limit 50 --read-only --no-input -o json

# Queries driving traffic to a specific page or path
gsc query -s <site> -d query -d page --last 28d --filter 'page ~ /blog/' --limit 500 --read-only --no-input -o json

# Exact page query breakdown
gsc query -s <site> -d query --last 28d --filter 'page = https://example.com/pricing' --read-only --no-input -o json
```

### 5. SEO Opportunity Insights

All `insights` subcommands compute opportunities locally from Search Analytics data:

```bash
# Keywords ranking positions 4-20 with meaningful impressions (quick wins)
gsc insights striking-distance -s <site> --last 28d --read-only --no-input -o json

# Queries with CTR far below the site's median at the same position (title/meta fixes)
gsc insights low-ctr -s <site> --last 28d --read-only --no-input -o json

# Queries split across multiple competing pages
gsc insights cannibalization -s <site> --last 3m --read-only --no-input -o json

# Queries or pages losing the most clicks vs previous period or YoY
gsc insights decliners -s <site> --last 28d --compare previous --by query --read-only --no-input -o json
gsc insights decliners -s <site> --last 28d --compare previous --by page --read-only --no-input -o json

# Newly appearing or disappeared queries/pages
gsc insights new-queries -s <site> --last 28d --read-only --no-input -o json
gsc insights lost-queries -s <site> --last 28d --read-only --no-input -o json
```

### 6. Sitemaps & Bulk Export

```bash
gsc sitemaps list -s <site> --read-only --no-input -o json
gsc export -s <site> --last 3m -d query -d page --out ./gsc-export --no-input
```

## Self-Discovery & References

- Run `gsc commands -o json` or `gsc <subcommand> --help` for exact flags at runtime.
- Full command flags, filter operators, and date rules: [references/commands.md](./references/commands.md)
- Authentication setup and error troubleshooting: [references/auth.md](./references/auth.md)
