package sitemap

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"reflect"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func gzipBytes(t *testing.T, raw string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFetchNestedAndGzipWithDedupe(t *testing.T) {
	indexXML := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://example.com/sitemap-posts.xml</loc></sitemap>
  <sitemap><loc>https://example.com/sitemap-pages.xml.gz</loc></sitemap>
  <sitemap><loc>https://example.com/sitemap_index.xml</loc></sitemap>
</sitemapindex>`
	postsXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/a</loc></url>
  <url><loc>https://example.com/b</loc></url>
</urlset>`
	pagesXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/b</loc></url>
  <url><loc>https://example.com/c</loc></url>
</urlset>`

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body []byte
		switch r.URL.String() {
		case "https://example.com/sitemap_index.xml":
			body = []byte(indexXML)
		case "https://example.com/sitemap-posts.xml":
			body = []byte(postsXML)
		case "https://example.com/sitemap-pages.xml.gz":
			body = gzipBytes(t, pagesXML)
		default:
			return &http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewReader(nil))}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
	})}

	got, err := Fetch(context.Background(), client, []string{"https://example.com/sitemap_index.xml"})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	want := []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
