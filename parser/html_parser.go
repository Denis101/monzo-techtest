package parser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type ParserOptions struct {
	Timeout           time.Duration
	SameSubdomain     bool
	Distinct          bool
	IgnoreFragments   bool
	IgnoredExtensions []string
	IgnoredPaths      []string
}

type Parser struct {
	client *http.Client
	opts   ParserOptions
}

type ParserOutput struct {
	Links      []string
	Status     string
	StatusCode int
}

type SimpleHttpResponse struct {
	Body       io.Reader
	Status     string
	StatusCode int
	Header     http.Header
}

func SanitiseUrl(rawUrl string) (string, error) {
	url, _, err := getUrl(rawUrl)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s://%s%s", url.Scheme, url.Host, strings.TrimSuffix(url.Path, "/")), nil
}

func NewParser(opts ParserOptions) *Parser {
	return &Parser{
		client: http.DefaultClient,
		opts:   opts,
	}
}

func (p *Parser) ParseLinks(input string) (ParserOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.opts.Timeout)
	defer cancel()

	url, baseUrl, err := getUrl(input)
	if err != nil {
		return ParserOutput{}, err
	}

	response, err := p.get(ctx, *url)
	if err != nil {
		return ParserOutput{}, err
	}

	links, err := parseLinksFromHtmlBody(response.Body)
	if err != nil {
		return ParserOutput{Status: response.Status, StatusCode: response.StatusCode}, err
	}

	return ParserOutput{
		Links:      p.filterLinks(links, baseUrl),
		Status:     response.Status,
		StatusCode: response.StatusCode,
	}, err
}

func (p *Parser) get(ctx context.Context, url url.URL) (SimpleHttpResponse, error) {
	res, err := p.handleRequest(ctx, url)
	if err != nil {
		return SimpleHttpResponse{}, err
	}

	if res.StatusCode == 301 || res.StatusCode == 302 {
		redirectUrl, err := url.Parse(res.Header.Get("Location"))
		if err != nil {
			return SimpleHttpResponse{}, err
		}

		redirectRes, err := p.handleRequest(ctx, *redirectUrl)
		if err != nil {
			return SimpleHttpResponse{}, err
		}

		return redirectRes, nil
	}

	return res, nil
}

func (p *Parser) handleRequest(ctx context.Context, url url.URL) (SimpleHttpResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return SimpleHttpResponse{}, err
	}

	res, err := p.client.Do(req)
	if err != nil {
		return SimpleHttpResponse{}, err
	}

	return SimpleHttpResponse{
		Body:       res.Body,
		Status:     res.Status,
		StatusCode: res.StatusCode,
		Header:     res.Header,
	}, nil
}

func (p *Parser) filterLinks(links []string, baseUrl string) []string {
	var filteredLinks []string
loop:
	for _, l := range links {
		if p.opts.IgnoreFragments && strings.Contains(l, "#") {
			continue
		}

		if len(p.opts.IgnoredExtensions) > 0 {
			for _, ext := range p.opts.IgnoredExtensions {
				if strings.HasSuffix(l, ext) {
					continue loop
				}
			}
		}

		if len(p.opts.IgnoredPaths) > 0 {
			for _, path := range p.opts.IgnoredPaths {
				if strings.Contains(l, path) {
					continue loop
				}
			}
		}

		if strings.HasPrefix(l, "/") {
			l = fmt.Sprintf("%s%s", baseUrl, l)
		}

		if p.opts.SameSubdomain && !strings.HasPrefix(l, baseUrl) {
			continue
		}

		sanitisedLink, err := SanitiseUrl(l)
		if err != nil {
			continue
		}

		filteredLinks = append(filteredLinks, sanitisedLink)
	}

	if p.opts.Distinct {
		filteredLinks = distinctLinks(filteredLinks)
	}

	return filteredLinks
}

func distinctLinks(links []string) []string {
	linkSet := make(map[string]bool)
	for _, l := range links {
		_, ok := linkSet[l]
		if !ok {
			linkSet[l] = true
		}
	}

	var distinctLinks []string
	for k := range linkSet {
		distinctLinks = append(distinctLinks, k)
	}

	return distinctLinks
}

func getUrl(rawUrl string) (*url.URL, string, error) {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, "", err
	}

	if parsedUrl.String() == "" {
		return nil, "", fmt.Errorf("empty url for input %s", rawUrl)
	}

	if parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https" {
		return nil, "", fmt.Errorf("missing or invalid scheme for input %s", rawUrl)
	}

	if parsedUrl.Host == "" {
		return nil, "", fmt.Errorf("missing host for input %s", rawUrl)
	}

	return parsedUrl, fmt.Sprintf("%s://%s", parsedUrl.Scheme, parsedUrl.Host), nil
}

func parseLinksFromHtmlBody(reader io.Reader) ([]string, error) {
	var links []string
	tokenizer := html.NewTokenizer(reader)

	for {
		tokenType := tokenizer.Next()

		switch {
		case tokenType == html.ErrorToken:
			if err := tokenizer.Err(); err != io.EOF {
				return nil, err
			}

			return links, nil
		case tokenType == html.StartTagToken:
			t := tokenizer.Token()
			if t.Data == "a" {
				for _, a := range t.Attr {
					if a.Key == "href" {
						links = append(links, a.Val)
					}
				}
			}
		}
	}
}
