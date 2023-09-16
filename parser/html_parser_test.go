package parser

import (
	"net/http"
	"strings"
	"testing"
)

func TestSanitiseUrlEmpty(t *testing.T) {
	_, err := SanitiseUrl("")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitiseUrlMissingScheme(t *testing.T) {
	_, err := SanitiseUrl("monzo.com")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitiseUrlInvalidScheme(t *testing.T) {
	_, err := SanitiseUrl("://monzo.com")
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = SanitiseUrl("bad://monzo.com")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitiseUrlMissingHost(t *testing.T) {
	_, err := SanitiseUrl("http://")
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = SanitiseUrl("https://")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitiseUrlRemovesTrailingSlash(t *testing.T) {
	expected := "https://monzo.com"

	url, err := SanitiseUrl("https://monzo.com/")
	if err != nil {
		t.Fatal("unexpected error")
	}

	if url != expected {
		t.Fatalf("expected: %s, actual: %s", expected, url)
	}
}

func TestParseLinksInvalidUrl(t *testing.T) {

	_, err := getDefaultTestParser().ParseLinks("monzo")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFilterLinksRelativePaths(t *testing.T) {
	baseUrl := "https://monzo.com"
	links := []string{
		"https://monzo.com/blog",
		"/about",
		"/blog/2023/something",
	}

	result := getDefaultTestParser().filterLinks(links, baseUrl)

	for _, l := range result {
		if !strings.HasPrefix(l, baseUrl) {
			t.Fatalf("%s missing baseUrl prefix %s", l, baseUrl)
		}
	}
}

func TestFilterLinksSameSubdomain(t *testing.T) {
	links := []string{
		"https://monzo.com/about",
		"https://instagram.com/monzo",
	}

	result := getTestParser(ParserOptions{SameSubdomain: true}).filterLinks(links, "https://monzo.com")
	if len(result) != 1 {
		t.Fatalf("expected len: %d, actual len: %d", 1, len(result))
	}

	result = getTestParser(ParserOptions{SameSubdomain: false}).filterLinks(links, "https://monzo.com")
	if len(result) != 2 {
		t.Fatalf("expected len: %d, actual len: %d", 2, len(result))
	}
}

func TestFilterLinksIgnoreFragments(t *testing.T) {
	links := []string{
		"https://monzo.com/about/",
		"https://monzo.com/about#fragment",
	}

	result := getTestParser(ParserOptions{IgnoreFragments: true}).filterLinks(links, "https://monzo.com")
	if len(result) != 1 {
		t.Fatalf("expected len: %d, actual len: %d", 1, len(result))
	}

	result = getTestParser(ParserOptions{IgnoreFragments: false}).filterLinks(links, "https://monzo.com")
	if len(result) != 2 {
		t.Fatalf("expected len: %d, actual len: %d", 2, len(result))
	}
}

func TestFilterLinksIgnoredExtensions(t *testing.T) {
	links := []string{
		"https://monzo.com/static/style.css",
		"https://monzo.com/static/credit-card.jpg",
		"https://monzo.com/static/scary-legal-document.pdf",
	}

	result := getTestParser(ParserOptions{IgnoredExtensions: []string{".css"}}).
		filterLinks(links, "https://monzo.com")
	if len(result) != 2 {
		t.Fatalf("expected len: %d, actual len: %d", 2, len(result))
	}

	result = getTestParser(ParserOptions{IgnoredExtensions: []string{".jpg"}}).
		filterLinks(links, "https://monzo.com")
	if len(result) != 2 {
		t.Fatalf("expected len: %d, actual len: %d", 2, len(result))
	}

	result = getTestParser(ParserOptions{IgnoredExtensions: []string{".pdf"}}).
		filterLinks(links, "https://monzo.com")
	if len(result) != 2 {
		t.Fatalf("expected len: %d, actual len: %d", 2, len(result))
	}

	result = getTestParser(ParserOptions{IgnoredExtensions: []string{".css", ".jpg"}}).
		filterLinks(links, "https://monzo.com")
	if len(result) != 1 {
		t.Fatalf("expected len: %d, actual len: %d", 1, len(result))
	}

	result = getTestParser(ParserOptions{IgnoredExtensions: []string{".css", ".jpg", ".pdf"}}).
		filterLinks(links, "https://monzo.com")
	if len(result) != 0 {
		t.Fatalf("expected len: %d, actual len: %d", 0, len(result))
	}
}

func TestFilterLinksDistinct(t *testing.T) {
	links := []string{
		"https://monzo.com/about",
		"https://monzo.com/about/",
		"/about",
		"/about/",
	}

	result := getTestParser(ParserOptions{Distinct: true}).filterLinks(links, "https://monzo.com")
	if len(result) != 1 {
		t.Fatalf("expected len: %d, actual len: %d", 1, len(result))
	}
}

func getDefaultTestParser() *Parser {
	return getTestParser(ParserOptions{})
}

func getTestParser(opts ParserOptions) *Parser {
	parser := NewParser(opts)
	parser.client = new(http.Client)
	return parser
}
