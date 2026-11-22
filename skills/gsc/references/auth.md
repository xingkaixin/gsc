# `gsc` Authentication & Troubleshooting

## Credential Precedence

1. `--access-token` / `GSC_ACCESS_TOKEN`
2. `--credentials` / `GSC_CREDENTIALS` (service account, authorized user, or external account JSON)
3. Selected profile in `~/.config/gsc/config.yaml` (`--profile` > `GSC_PROFILE` > `current_profile`)

`gsc` never silently falls back to ambient ADC credentials.

## Recommended Setup: Service Account

Best for personal long-term use, scripts, and agents (no browser prompt, no 7-day OAuth expiry):

1. Enable Google Search Console API in Google Cloud.
2. Create a Service Account and download its JSON key to `~/.config/gsc/service-account.json` (`chmod 600`).
3. Add the Service Account email in Google Search Console under **Settings → Users and permissions** (Restricted for read-only; Full if submitting sitemaps).
4. Save the profile:
   ```bash
   gsc auth login --service-account ~/.config/gsc/service-account.json
   gsc sites list
   gsc sites use sc-domain:example.com
   ```

## Alternative: Own Desktop OAuth Client

Needed if you want browser login across all properties under your Google account without adding a service account per site:

```bash
gsc auth login --client-secret-file /path/to/client_secret.json
```

Keep the OAuth consent screen in **Production** (Testing expires refresh tokens every 7 days).

## Common Errors

| Error / Kind | Likely Cause & Fix |
| --- | --- |
| `not logged in` (`auth`) | Run `gsc auth login --service-account <key.json>` or set `GSC_CREDENTIALS`. |
| `HTTP 403` permission denied (`permission`) | Service account or Google account is not a member of the target property in Search Console → Settings → Users and permissions. |
| `HTTP 403` `accessNotConfigured` | Enable `searchconsole.googleapis.com` in the credential's Google Cloud project. |
| `no site given` | Pass `-s <site>`, set `GSC_SITE`, or run `gsc sites use <site>`. |
| `rate_limit` / quota exhausted | URL Inspection quota is 2,000/day & 600/min per property; Search Analytics has load quotas on wide date ranges with `query` + `page` together. Narrow dates or wait ~15 minutes. |
| `no sitemaps found` on `index-audit` | Submit a sitemap in Search Console or pass `--sitemap https://example.com/sitemap.xml` explicitly. |
