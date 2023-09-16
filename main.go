package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/denis101/monzo-techtest/crawler"
	hclog "github.com/hashicorp/go-hclog"
)

var debugLogFlag = flag.Bool("v", false, "Enable DEBUG level logging")
var traceLogFlag = flag.Bool("vv", false, "Enable TRACE level logging")
var logJsonFlag = flag.Bool("json-log", false, "Enable json logging")

var urlFlag = flag.String("url", "https://crawler-test.com/", "URL to crawl")
var outputFlag = flag.String("o", "", "Output filename")
var formatFlag = flag.String("f", "stdout", "Output format [stdout|json|xml]")
var interactiveFlag = flag.Bool("i", false, "Interactive mode")
var maxWorkersFlag = flag.Int("workers", 2, "Amount of worker threads")
var deadlineFlag = flag.Int("deadline", 5, "HTTP request deadline in seconds")
var ignoreFragmentsFlag = flag.Bool("fragments", true, "Ignore URLs with fragments in their paths")
var ignoredExtensionsFlag = flag.String("ext", "", "Ignore URLs ending in the provided extensions (e.g. .jpg)")
var ignoredPathsFlag = flag.String("paths", "", "Ignore URLs containing the provided strings in their paths")

func main() {
	flag.Parse()
	logLevel := "info"
	if *debugLogFlag {
		logLevel = "debug"
	}
	if *traceLogFlag {
		logLevel = "trace"
	}

	hclog.SetDefault(hclog.New(&hclog.LoggerOptions{
		Level:      hclog.LevelFromString(logLevel),
		JSONFormat: *logJsonFlag,
	}))

	if !strings.Contains(*urlFlag, "http") {
		panic(fmt.Errorf("client error: invalid parameter url, missing scheme in [%s]", *urlFlag))
	}

	if *formatFlag != string(crawler.Output_Stdout) && *formatFlag != string(crawler.Output_Json) && *formatFlag != string(crawler.Output_Xml) {
		panic(fmt.Errorf("client error: invalid parameter o, unsupported format [%s]", *formatFlag))
	}

	var ignoredExtensions []string
	if len(*ignoredExtensionsFlag) > 0 {
		ignoredExtensions = strings.Split(*ignoredExtensionsFlag, ",")
	}

	var ignoredPaths []string
	if len(*ignoredExtensionsFlag) > 0 {
		ignoredPaths = strings.Split(*ignoredPathsFlag, ",")
	}

	crawler.NewCrawler(crawler.CrawlerOptions{
		MaxWorkers:        *maxWorkersFlag,
		OutputFormat:      crawler.CrawlerOutputFormat(*formatFlag),
		OutputFile:        *outputFlag,
		Interactive:       *interactiveFlag,
		RequestDeadline:   *deadlineFlag,
		IgnoreFragments:   *ignoreFragmentsFlag,
		IgnoredExtensions: ignoredExtensions,
		IgnoredPaths:      ignoredPaths,
	}).Crawl(*urlFlag)
}
