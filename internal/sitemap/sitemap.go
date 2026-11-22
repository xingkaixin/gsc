package sitemap

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	maxBodyBytes = 32 << 20
	maxDepth     = 5
	maxSitemaps  = 200
)

type urlSet struct {
	URLs []struct {
		Loc string `xml:"loc"`
	} `xml:"url"`
}

type sitemapIndex struct {
	Sitemaps []struct {
		Loc string `xml:"loc"`
	} `xml:"sitemap"`
}

func Fetch(ctx context.Context, httpClient *http.Client, roots []string) ([]string, error) {
	if len(roots) == 0 {
		return nil, errors.New("no sitemap URLs provided")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	visited := map[string]bool{}
	seenURL := map[string]bool{}
	var urls []string
	var walk func(u string, depth int) error
	walk = func(u string, depth int) error {
		u = strings.TrimSpace(u)
		if u == "" || visited[u] {
			return nil
		}
		if depth > maxDepth {
			return fmt.Errorf("sitemap nesting exceeds max depth (%d) at %s", maxDepth, u)
		}
		if len(visited) >= maxSitemaps {
			return fmt.Errorf("sitemap count exceeds limit (%d)", maxSitemaps)
		}
		visited[u] = true

		childMaps, pageURLs, err := fetchOne(ctx, httpClient, u)
		if err != nil {
			return err
		}
		for _, page := range pageURLs {
			if !seenURL[page] {
				seenURL[page] = true
				urls = append(urls, page)
			}
		}
		for _, child := range childMaps {
			if err := walk(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range roots {
		if err := walk(root, 0); err != nil {
			return nil, err
		}
	}
	return urls, nil
}

func fetchOne(ctx context.Context, httpClient *http.Client, rawURL string) (childMaps, pageURLs []string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid sitemap URL %q: %w", rawURL, err)
	}
	req.Header.Set("User-Agent", "gsc")
	req.Header.Set("Accept", "application/xml, text/xml, application/gzip, */*")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch sitemap %s: %w", rawURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("fetch sitemap %s: HTTP %d", rawURL, res.StatusCode)
	}

	br := bufio.NewReader(io.LimitReader(res.Body, maxBodyBytes+1))
	var r io.Reader = br
	// Peek gzip magic bytes so both .xml.gz and gzip-encoded responses work.
	if head, perr := br.Peek(2); perr == nil && head[0] == 0x1f && head[1] == 0x8b {
		gz, gerr := gzip.NewReader(br)
		if gerr != nil {
			return nil, nil, fmt.Errorf("gunzip sitemap %s: %w", rawURL, gerr)
		}
		defer gz.Close()
		r = io.LimitReader(gz, maxBodyBytes+1)
	}

	body, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("read sitemap %s: %w", rawURL, err)
	}
	if len(body) > maxBodyBytes {
		return nil, nil, fmt.Errorf("sitemap %s exceeds 32 MiB", rawURL)
	}
	return parseXML(rawURL, body)
}

func parseXML(source string, body []byte) (childMaps, pageURLs []string, err error) {
	dec := xml.NewDecoder(strings.NewReader(string(body)))
	for {
		tok, terr := dec.Token()
		if errors.Is(terr, io.EOF) {
			return nil, nil, fmt.Errorf("sitemap %s has no root XML element", source)
		}
		if terr != nil {
			return nil, nil, fmt.Errorf("parse sitemap %s: %w", source, terr)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "sitemapindex":
			var idx sitemapIndex
			if err := dec.DecodeElement(&idx, &start); err != nil {
				return nil, nil, fmt.Errorf("decode sitemapindex %s: %w", source, err)
			}
			for _, sm := range idx.Sitemaps {
				if loc := strings.TrimSpace(sm.Loc); isHTTPURL(loc) {
					childMaps = append(childMaps, loc)
				}
			}
			return childMaps, nil, nil
		case "urlset":
			var set urlSet
			if err := dec.DecodeElement(&set, &start); err != nil {
				return nil, nil, fmt.Errorf("decode urlset %s: %w", source, err)
			}
			for _, u := range set.URLs {
				if loc := strings.TrimSpace(u.Loc); isHTTPURL(loc) {
					pageURLs = append(pageURLs, loc)
				}
			}
			return nil, pageURLs, nil
		default:
			return nil, nil, fmt.Errorf("sitemap %s has unexpected root element <%s>", source, start.Name.Local)
		}
	}
}

func isHTTPURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}
