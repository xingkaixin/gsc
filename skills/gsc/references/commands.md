# `gsc` Command Reference

## Global Flags

| Flag | Env Var | Purpose |
| --- | --- | --- |
| `-s, --site` | `GSC_SITE` | Property (`sc-domain:example.com`, `https://example.com/`, or bare host `example.com`) |
| `--profile` | `GSC_PROFILE` | Named profile in `~/.config/gsc/config.yaml` |
| `-o, --output` | — | Output format: `table` (default), `json`, `yaml`, `csv` |
| `--read-only` | `GSC_READ_ONLY` | Block all remote writes, local config/auth writes, and `auth token` |
| `--no-input` | `GSC_NO_INPUT` | Never prompt interactively or open a browser |
| `--dry-run` | — | Print remote write request without sending it |
| `-y, --yes` | — | Confirm destructive commands without prompting |
| `-q, --quiet` | `GSC_QUIET` | Suppress informational stderr lines |
| `-v, --verbose` | `GSC_VERBOSE` | Log HTTP method, URL, and status to stderr |
| `--timeout` | — | HTTP timeout (default `30s`) |
| `--credentials` | `GSC_CREDENTIALS` | One-off Google credentials JSON file |
| `--access-token` | `GSC_ACCESS_TOKEN` | One-off OAuth bearer token (highest precedence) |

## Read Commands

### Properties & Diagnostics
- `gsc status [--verify]` / `gsc auth status [--verify]` — active profile, auth type, default site.
- `gsc doctor [--online]` — verify config, key file, PATH, and optional API call.
- `gsc sites list` — list accessible properties and permission levels.
- `gsc sites get [SITE]` — fetch one property's metadata.
- `gsc sitemaps list [--index URL]` — list sitemaps registered in Search Console.
- `gsc sitemaps get <FEED_URL>` — inspect one registered sitemap's status, errors, and warnings.
- `gsc freshness [--type web|image|video|news|discover|googleNews]` — latest final and preliminary dates.
- `gsc commands` / `gsc commands resolve -- <args>` — offline command catalog and parser dry-check.
- `gsc api methods` — offline table of all Search Console API v1 methods.

### Indexing & URL Inspection
- `gsc index-audit` — fetch sitemap XML(s) (auto-discovered from GSC or via `--sitemap`, supports `sitemapindex` and `.xml.gz`), inspect URLs, and summarize indexed/unindexed counts and `coverage_state` reasons.
  - `--sitemap <URL>` (repeatable)
  - `--prefix <URL_PREFIX>`
  - `--match <RE2_REGEX>`
  - `--only-unindexed` — only include unindexed or errored URLs in `results`
  - `--urls-only` — list discovered/matched URLs without calling URL Inspection API
  - `--max <N>` (default `2000`), `--concurrency <1-10>` (default `5`), `--interval <DUR>` (default `120ms`), `--language <BCP47>`
- `gsc inspect [URL...]` — inspect specific URLs directly.
  - `--file <PATH|- >` (one URL per line, `-` for stdin), `--max`, `--interval`, `--language`.

### Search Analytics
- `gsc performance` — period totals (`clicks`, `impressions`, `ctr`, `position`) with no dimensions.
- `gsc top queries|pages|countries|devices|appearance` — top 25 rows by clicks (override with `--limit` or `--all`).
- `gsc trend [--by date|hour|week|month]` — time series (`week`/`month` rolled up locally from daily rows; `hour` uses `--data-state hourly_all`).
- `gsc query` — full Search Analytics query.
  - `-d, --dimension` (repeatable): `query`, `page`, `date`, `hour`, `country`, `device`, `searchAppearance`
  - `--type`: `web` (default), `image`, `video`, `news`, `discover`, `googleNews`
  - `--data-state`: `final` (default), `all` (includes preliminary), `hourly_all`
  - `--aggregation`: `auto`, `byProperty`, `byPage`, `byNewsShowcasePanel`
  - `--filter 'DIM OP VALUE'` (repeatable, ANDed):
    - `=` equals (case-sensitive for `query` and `page`)
    - `!=` not equals
    - `~` contains
    - `!~` not contains
    - `=~` matches RE2 regex
    - `!=~` does not match RE2 regex
  - Date range: `--last 7d|28d|3m|16m` (optional `--end YYYY-MM-DD`) OR `--start YYYY-MM-DD [--end YYYY-MM-DD]`
  - Comparison: `--compare previous|yoy`
  - Pagination: `--limit <N>` (default `1000`, `25` for `top`) or `--all`
  - `--print-request` — print normalized API payload without calling the API

### SEO Insights
- `gsc insights striking-distance` — `--min-position` (4), `--max-position` (20), `--min-impressions` (100), `--limit` (50)
- `gsc insights low-ctr` — `--min-impressions` (100), `--factor` (0.5), `--limit` (50)
- `gsc insights cannibalization` — `--min-share` (0.1), `--min-impressions` (100), `--limit` (25)
- `gsc insights decliners` — `--by query|page`, `--compare previous|yoy`, `--min-clicks`, `--limit`
- `gsc insights new-queries` — `--by query|page`, `--compare previous|yoy`, `--limit`
- `gsc insights lost-queries` — `--by query|page`, `--compare previous|yoy`, `--limit`

## Write Commands (Require Explicit User Confirmation)

- Local writes: `gsc auth login`, `gsc auth use`, `gsc auth logout`, `gsc sites use`, `gsc config use-profile`, `gsc export`
- Remote writes: `gsc sites add`, `gsc sites remove`, `gsc sitemaps submit`, `gsc sitemaps delete`, `gsc api <POST|PUT|DELETE> <PATH>`
