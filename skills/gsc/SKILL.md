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
2. **Keep outputs clean, small, and non-truncated.**
   - Run each `gsc` command as its own parallel tool call rather than chaining multiple `-o json` commands with `;` in one shell call (chaining mixes stderr lines between JSON blocks and causes large truncated outputs).
   - Prefer `--read-only --no-input -q -o json` for read queries so `stdout` is pure JSON with `site`, date ranges, and structured fields.
   - Never run mutating commands (`sites add/remove`, `sitemaps submit/delete`, `auth logout`) without explicit user confirmation; preview remote writes with `--dry-run` first.
   - Never run `gsc auth token` or read/print credential JSON files (`service-account.json`, `secrets.yaml`).
3. **Reuse `--end` after `freshness` to halve API round-trips.**
   - Run `gsc freshness -s <site>` once to get `latest_final_date` (Search Analytics dates are in Pacific Time and typically lag 2–3 days).
   - Pass `--end <latest_final_date>` on subsequent `--last` queries in the same check so each command skips redundant freshness probes.
4. **Run `index-audit` decisively with quota thresholds.**
   - Always check `gsc index-audit -s <site> --urls-only --read-only --no-input -q -o json` first (0 inspection quota; daily limit is 2,000/property).
   - If `matched <= 300`: run the full audit immediately (`gsc index-audit -s <site> --only-unindexed --read-only --no-input -q -o json`) instead of manually sampling a few URLs—it runs concurrently and finishes in seconds.
   - If `matched > 300`: scope with `--prefix`, `--match`, or `--max` (e.g., audit key sections like `/blog/` or `/docs/` first).
5. **Adapt `insights` thresholds for low-traffic sites and respect API math.**
   - `insights` defaults to `--min-impressions 100`. If `gsc performance` shows low total impressions (e.g. `< 5,000` over 28 days), lower `--min-impressions` (e.g. `5` or `10`) when calling `insights`, or inspect `gsc query -d query -d page` directly so you don't get empty results.
   - Never sum rows from `top queries` or `query` to compute site totals—Google withholds anonymized queries. Always use `gsc performance` for totals.
   - Google has no public API for the full Page Indexing report (pages outside sitemaps are not discoverable via API), and `inspect` returns the last indexed snapshot—not a live test or "Request indexing".

## Core SEO Workflows

### 1. Check Setup, Properties & Data Freshness

```bash
gsc status -o json
gsc sites list --read-only --no-input -q -o json
gsc freshness -s <site> --read-only --no-input -q -o json
```

### 2. Indexing & Unindexed Coverage Audit

```bash
# Step 1: Preview sitemap URL count (0 inspection quota)
gsc index-audit -s <site> --urls-only --read-only --no-input -q -o json

# Step 2: If matched <= 300, audit all sitemap URLs and return unindexed/errored pages + grouped reasons
gsc index-audit -s <site> --only-unindexed --read-only --no-input -q -o json

# If matched > 300, filter to a section or cap with --max
gsc index-audit -s <site> --prefix https://example.com/blog/ --only-unindexed --read-only --no-input -q -o json

# Inspect specific URLs directly
gsc inspect -s <site> https://example.com/pricing https://example.com/blog/post --read-only --no-input -q -o json
```

When reporting `index-audit` results to the user, lead with:
- Total discovered vs. inspected URLs, indexed count & index rate (`indexed / inspected`).
- Top `unindexed_reasons` grouped by `coverage_state` (e.g., `Discovered - currently not indexed`, `Crawled - currently not indexed`).
- Concrete sample URLs for the main unindexed causes.

### 3. Overall Performance, Trends & Audience Breakdown

Pass `--end <latest_final_date>` from `gsc freshness` on each call:

```bash
# Site totals for last 28 days compared to previous 28 days
gsc performance -s <site> --last 28d --end <YYYY-MM-DD> --compare previous --read-only --no-input -q -o json

# Daily / weekly / monthly trend
gsc trend -s <site> --last 3m --end <YYYY-MM-DD> --by week --read-only --no-input -q -o json

# Country and device distribution (prefer `top` over raw `query`)
gsc top countries -s <site> --last 28d --end <YYYY-MM-DD> --limit 10 --read-only --no-input -q -o json
gsc top devices -s <site> --last 28d --end <YYYY-MM-DD> --read-only --no-input -q -o json
```

### 4. Matched Keywords & Top Pages

```bash
# Top matched keywords (queries)
gsc top queries -s <site> --last 28d --end <YYYY-MM-DD> --limit 50 --compare previous --read-only --no-input -q -o json

# Top landing pages
gsc top pages -s <site> --last 28d --end <YYYY-MM-DD> --limit 50 --read-only --no-input -q -o json

# Queries driving traffic to a specific path or page
gsc query -s <site> -d query -d page --last 28d --end <YYYY-MM-DD> --filter 'page ~ /blog/' --limit 200 --read-only --no-input -q -o json
```

### 5. SEO Opportunity Insights

Lower `--min-impressions` (e.g. `5` or `10`) for low-traffic sites (`< 5,000` impressions in 28d):

```bash
# Keywords ranking positions 4-20 with meaningful impressions (quick wins)
gsc insights striking-distance -s <site> --last 28d --end <YYYY-MM-DD> --min-impressions 10 --read-only --no-input -q -o json

# Queries with CTR far below the site's median at the same position (title/meta fixes)
gsc insights low-ctr -s <site> --last 28d --end <YYYY-MM-DD> --min-impressions 10 --read-only --no-input -q -o json

# Queries split across multiple competing pages
gsc insights cannibalization -s <site> --last 3m --end <YYYY-MM-DD> --min-impressions 10 --read-only --no-input -q -o json

# Queries or pages losing the most clicks vs previous period or YoY
gsc insights decliners -s <site> --last 28d --end <YYYY-MM-DD> --compare previous --by query --read-only --no-input -q -o json

# Newly appearing or disappeared queries/pages
gsc insights new-queries -s <site> --last 28d --end <YYYY-MM-DD> --read-only --no-input -q -o json
gsc insights lost-queries -s <site> --last 28d --end <YYYY-MM-DD> --read-only --no-input -q -o json
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
