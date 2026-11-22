package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/xingkaixin/gsc/internal/client"
	"github.com/xingkaixin/gsc/internal/output"
	"github.com/xingkaixin/gsc/internal/sitemap"
	"github.com/spf13/cobra"
)

type reasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type auditItem struct {
	URL             string `json:"url"`
	Indexed         bool   `json:"indexed"`
	Verdict         string `json:"verdict,omitempty"`
	CoverageState   string `json:"coverage_state,omitempty"`
	IndexingState   string `json:"indexing_state,omitempty"`
	LastCrawlTime   string `json:"last_crawl_time,omitempty"`
	GoogleCanonical string `json:"google_canonical,omitempty"`
	Error           string `json:"error,omitempty"`
}

type indexAuditReport struct {
	Site       string        `json:"site"`
	Sitemaps   []string      `json:"sitemaps"`
	Discovered int           `json:"discovered"`
	Matched    int           `json:"matched"`
	Requested  int           `json:"requested"`
	Inspected  int           `json:"inspected"`
	Indexed    int           `json:"indexed"`
	NotIndexed int           `json:"not_indexed"`
	Errors     int           `json:"errors"`
	IndexRate  float64       `json:"index_rate"`
	Reasons    []reasonCount `json:"unindexed_reasons"`
	Stopped    string        `json:"stopped_reason,omitempty"`
	Results    []auditItem   `json:"results"`
}

func (r *indexAuditReport) Table() output.Table {
	t := output.Table{
		Columns:      []string{"url", "indexed", "verdict", "coverage_state", "indexing_state", "last_crawl_time", "google_canonical", "error"},
		HumanColumns: []string{"url", "indexed", "verdict", "coverage_state", "last_crawl_time", "error"},
		Human:        map[string]func(any) string{"last_crawl_time": output.Timestamp},
	}
	for _, item := range r.Results {
		t.Rows = append(t.Rows, []any{
			item.URL,
			item.Indexed,
			nullIfEmpty(item.Verdict),
			nullIfEmpty(item.CoverageState),
			nullIfEmpty(item.IndexingState),
			nullIfEmpty(item.LastCrawlTime),
			nullIfEmpty(item.GoogleCanonical),
			nullIfEmpty(item.Error),
		})
	}
	return t
}

type auditURLList struct {
	Site       string   `json:"site"`
	Sitemaps   []string `json:"sitemaps"`
	Discovered int      `json:"discovered"`
	Matched    int      `json:"matched"`
	URLs       []string `json:"urls"`
}

func (l *auditURLList) Table() output.Table {
	t := output.Table{Columns: []string{"url"}}
	for _, u := range l.URLs {
		t.Rows = append(t.Rows, []any{u})
	}
	return t
}

func (a *app) indexAudit() *cobra.Command {
	var sitemaps []string
	var prefix, matchPattern, language string
	var maxURLs int
	var interval time.Duration
	var onlyUnindexed, urlsOnly bool

	c := &cobra.Command{
		Use:   "index-audit",
		Short: "Audit indexing status across sitemap URLs and summarize unindexed reasons",
		Long: "Fetches sitemap XMLs (from --sitemap or sitemaps submitted in Search Console), inspects each URL\n" +
			"with the URL Inspection API, and summarizes indexed vs. unindexed counts and coverage reasons.\n" +
			"Use --urls-only to preview discovered URLs without consuming inspection quota (2,000/day per site).",
		Example: "  gsc index-audit\n" +
			"  gsc index-audit --only-unindexed\n" +
			"  gsc index-audit --sitemap https://example.com/sitemap.xml --prefix https://example.com/blog/\n" +
			"  gsc index-audit --urls-only",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if maxURLs <= 0 {
				return withKind(kindUsage, errors.New("--max must be positive"))
			}
			var re *regexp.Regexp
			if matchPattern != "" {
				var err error
				re, err = regexp.Compile(matchPattern)
				if err != nil {
					return withKind(kindUsage, fmt.Errorf("invalid --match regex: %w", err))
				}
			}

			cl, _, err := a.connect(ctx)
			if err != nil {
				return err
			}
			site, err := a.resolveSite(ctx, cl, "")
			if err != nil {
				return err
			}

			sources := append([]string{}, sitemaps...)
			if len(sources) == 0 {
				listed, err := cl.ListSitemaps(ctx, site, "")
				if err != nil {
					return err
				}
				for _, entry := range listed {
					if p, ok := entry["path"].(string); ok && strings.TrimSpace(p) != "" {
						sources = append(sources, strings.TrimSpace(p))
					}
				}
				sources = dedupe(sources)
				if len(sources) == 0 {
					fallback := defaultSitemapURL(site)
					if fallback == "" {
						return withKind(kindUsage, errors.New("no sitemaps found in Search Console; pass --sitemap URL"))
					}
					a.info("No sitemaps registered in Search Console; trying %s", fallback)
					sources = []string{fallback}
				}
			}

			// Public sitemap requests should follow standard redirects without API credentials.
			fetchClient := &http.Client{Timeout: a.timeout, Transport: a.transport}
			discovered, err := sitemap.Fetch(ctx, fetchClient, sources)
			if err != nil {
				return err
			}
			if len(discovered) == 0 {
				return fmt.Errorf("no URLs found in sitemaps: %s", strings.Join(sources, ", "))
			}

			filtered := make([]string, 0, len(discovered))
			for _, u := range discovered {
				if prefix != "" && !strings.HasPrefix(u, prefix) {
					continue
				}
				if re != nil && !re.MatchString(u) {
					continue
				}
				filtered = append(filtered, u)
			}

			if urlsOnly {
				a.info("Discovered %d URLs (%d matched) from %d sitemap(s).", len(discovered), len(filtered), len(sources))
				return a.print(&auditURLList{
					Site:       site,
					Sitemaps:   sources,
					Discovered: len(discovered),
					Matched:    len(filtered),
					URLs:       filtered,
				})
			}

			targets := filtered
			if len(targets) > maxURLs {
				a.warn("%d URLs matched; inspecting the first %d (--max)", len(targets), maxURLs)
				targets = targets[:maxURLs]
			}

			report := &indexAuditReport{
				Site:       site,
				Sitemaps:   sources,
				Discovered: len(discovered),
				Matched:    len(filtered),
				Requested:  len(targets),
				Reasons:    []reasonCount{},
				Results:    []auditItem{},
			}

			reasonCounts := map[string]int{}
			var stopErr error
			for i, u := range targets {
				if i > 0 && interval > 0 {
					if err := a.wait(ctx, interval); err != nil {
						stopErr = err
						break
					}
				}
				res, err := cl.Inspect(ctx, u, site, language)
				if err != nil {
					var api *client.APIError
					if errors.As(err, &api) && (api.Quota() || api.Status == 401) || ctx.Err() != nil {
						report.Stopped = err.Error()
						stopErr = err
						break
					}
					report.Errors++
					report.Results = append(report.Results, auditItem{URL: u, Error: err.Error()})
					continue
				}

				item := buildAuditItem(u, res)
				report.Inspected++
				if item.Indexed {
					report.Indexed++
				} else {
					report.NotIndexed++
					reason := item.CoverageState
					if reason == "" {
						reason = item.Verdict
					}
					if reason == "" {
						reason = "Unknown"
					}
					reasonCounts[reason]++
				}
				if !onlyUnindexed || !item.Indexed {
					report.Results = append(report.Results, item)
				}
				if len(targets) > 1 && (i+1)%25 == 0 {
					a.info("Inspected %d of %d...", i+1, len(targets))
				}
			}

			if report.Inspected > 0 {
				report.IndexRate = float64(report.Indexed) * 100 / float64(report.Inspected)
			}
			for reason, count := range reasonCounts {
				report.Reasons = append(report.Reasons, reasonCount{Reason: reason, Count: count})
			}
			sort.Slice(report.Reasons, func(i, j int) bool {
				if report.Reasons[i].Count != report.Reasons[j].Count {
					return report.Reasons[i].Count > report.Reasons[j].Count
				}
				return report.Reasons[i].Reason < report.Reasons[j].Reason
			})

			if a.format != "json" && a.format != "yaml" {
				a.info("Site: %s | Discovered: %d | Inspected: %d | Indexed: %d (%.1f%%) | Not indexed: %d | Errors: %d",
					report.Site, report.Discovered, report.Inspected, report.Indexed, report.IndexRate, report.NotIndexed, report.Errors)
				if len(report.Reasons) > 0 {
					a.info("Unindexed reasons:")
					for _, rc := range report.Reasons {
						a.info("  - %s: %d", rc.Reason, rc.Count)
					}
				}
			}

			if err := a.print(report); err != nil {
				return err
			}
			if stopErr != nil {
				return fmt.Errorf("stopped after %d of %d URLs: %w", report.Inspected, len(targets), stopErr)
			}
			return nil
		},
	}

	c.Flags().StringArrayVar(&sitemaps, "sitemap", nil, "Sitemap XML URL to audit (repeatable; defaults to sitemaps listed in Search Console)")
	c.Flags().StringVar(&prefix, "prefix", "", "Only include sitemap URLs starting with this prefix")
	c.Flags().StringVar(&matchPattern, "match", "", "Only include sitemap URLs matching this regular expression")
	c.Flags().BoolVar(&onlyUnindexed, "only-unindexed", false, "Only include unindexed or errored URLs in the detailed rows")
	c.Flags().BoolVar(&urlsOnly, "urls-only", false, "Only list discovered sitemap URLs without calling the URL Inspection API")
	c.Flags().StringVar(&language, "language", "", "BCP 47 language for issue messages (default en-US)")
	c.Flags().IntVar(&maxURLs, "max", 2000, "Inspect at most this many URLs (the per-property daily quota is 2,000)")
	c.Flags().DurationVar(&interval, "interval", 120*time.Millisecond, "Pause between inspections (stays under 600 per minute)")
	return c
}

func buildAuditItem(u string, res map[string]any) auditItem {
	idx, _ := res["indexStatusResult"].(map[string]any)
	str := func(k string) string {
		if idx == nil {
			return ""
		}
		v, _ := idx[k].(string)
		return v
	}
	verdict := str("verdict")
	return auditItem{
		URL:             u,
		Indexed:         verdict == "PASS",
		Verdict:         verdict,
		CoverageState:   str("coverageState"),
		IndexingState:   str("indexingState"),
		LastCrawlTime:   str("lastCrawlTime"),
		GoogleCanonical: str("googleCanonical"),
	}
}

func defaultSitemapURL(site string) string {
	if strings.HasPrefix(site, "sc-domain:") {
		host := strings.TrimSpace(strings.TrimPrefix(site, "sc-domain:"))
		if host == "" {
			return ""
		}
		return "https://" + host + "/sitemap.xml"
	}
	u, err := url.Parse(site)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/sitemap.xml"
}
