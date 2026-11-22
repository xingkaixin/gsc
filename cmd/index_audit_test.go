package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type auditTransport struct {
	google   *fakeGoogle
	sitemaps map[string]string
}

func (t *auditTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if body, ok := t.sitemaps[r.URL.String()]; ok {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": {"application/xml"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		}, nil
	}
	return t.google.RoundTrip(r)
}

func runAuditCLI(t *testing.T, tr http.RoundTripper, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	a := newApp(strings.NewReader(""), &out, &errb)
	a.transport = tr
	a.now = func() time.Time { return now }
	a.sleep = func(context.Context, time.Duration) error { return nil }
	code := a.run(context.Background(), args)
	return result{code, out.String(), errb.String()}
}

func TestIndexAuditSummarizesReasonsAndFilters(t *testing.T) {
	isolate(t)
	f := newFake(t)
	f.inspect = func(u string) (int, string) {
		switch u {
		case "https://example.com/":
			return 200, `{"inspectionResult":{"indexStatusResult":{"verdict":"PASS","coverageState":"Submitted and indexed","indexingState":"INDEXING_ALLOWED","lastCrawlTime":"2026-10-01T00:00:00Z"}}}`
		case "https://example.com/blog/a":
			return 200, `{"inspectionResult":{"indexStatusResult":{"verdict":"NEUTRAL","coverageState":"Discovered - currently not indexed"}}}`
		case "https://example.com/blog/b":
			return 200, `{"inspectionResult":{"indexStatusResult":{"verdict":"NEUTRAL","coverageState":"Crawled - currently not indexed"}}}`
		default:
			return 200, `{"inspectionResult":{"indexStatusResult":{"verdict":"NEUTRAL","coverageState":"Discovered - currently not indexed"}}}`
		}
	}
	tr := &auditTransport{
		google: f,
		sitemaps: map[string]string{
			"https://example.com/sitemap.xml": `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/</loc></url>
  <url><loc>https://example.com/blog/a</loc></url>
  <url><loc>https://example.com/blog/b</loc></url>
  <url><loc>https://example.com/blog/c</loc></url>
</urlset>`,
		},
	}

	r := runAuditCLI(t, tr, "index-audit", "--site", "sc-domain:example.com", "--access-token", "tok", "--only-unindexed", "--interval", "0s", "-o", "json")
	if r.code != 0 {
		t.Fatalf("code=%d err=%s", r.code, r.errOut)
	}
	var report indexAuditReport
	if err := json.Unmarshal([]byte(r.out), &report); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, r.out)
	}
	if report.Discovered != 4 || report.Inspected != 4 || report.Indexed != 1 || report.NotIndexed != 3 {
		t.Fatalf("unexpected counts: %+v", report)
	}
	if len(report.Results) != 3 {
		t.Fatalf("expected only 3 unindexed rows, got %d", len(report.Results))
	}
	if len(report.Reasons) != 2 || report.Reasons[0].Reason != "Discovered - currently not indexed" || report.Reasons[0].Count != 2 {
		t.Fatalf("unexpected reasons: %+v", report.Reasons)
	}
}

func TestIndexAuditURLsOnlyConsumesNoInspectionQuota(t *testing.T) {
	isolate(t)
	f := newFake(t)
	tr := &auditTransport{
		google: f,
		sitemaps: map[string]string{
			"https://example.com/sitemap.xml": `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/</loc></url>
  <url><loc>https://example.com/blog/post-1</loc></url>
  <url><loc>https://example.com/docs/intro</loc></url>
</urlset>`,
		},
	}

	r := runAuditCLI(t, tr, "index-audit", "--site", "sc-domain:example.com", "--access-token", "tok", "--prefix", "https://example.com/blog/", "--urls-only", "-o", "json")
	if r.code != 0 {
		t.Fatalf("code=%d err=%s", r.code, r.errOut)
	}
	if f.count("POST searchconsole.googleapis.com/v1/urlInspection/index:inspect") != 0 {
		t.Fatalf("expected 0 inspect calls in --urls-only mode")
	}
	var out auditURLList
	if err := json.Unmarshal([]byte(r.out), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Discovered != 3 || out.Matched != 1 || len(out.URLs) != 1 || !strings.HasSuffix(out.URLs[0], "/blog/post-1") {
		t.Fatalf("unexpected output: %+v", out)
	}
}
