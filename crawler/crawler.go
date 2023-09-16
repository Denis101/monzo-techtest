package crawler

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/denis101/monzo-techtest/parser"
	"github.com/denis101/monzo-techtest/scheduler"
	"github.com/hashicorp/go-hclog"
	"github.com/pterm/pterm"

	"github.com/fatih/structs"
)

type CrawlerOutputFormat string

const (
	Output_Stdout CrawlerOutputFormat = "stdout"
	Output_Json   CrawlerOutputFormat = "json"
	Output_Xml    CrawlerOutputFormat = "xml"
)

const UpdateDuration = time.Millisecond * 200

var SpinnerSequence []string = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

type CrawlerOptions struct {
	OutputFormat      CrawlerOutputFormat `structs:",omitempty"`
	OutputFile        string              `structs:",omitempty"`
	MaxWorkers        int
	Interactive       bool
	RequestDeadline   int
	IgnoreFragments   bool
	IgnoredExtensions []string `structs:",omitempty"`
	IgnoredPaths      []string `structs:",omitempty"`
}

type Crawler struct {
	scheduler *scheduler.Scheduler[string]
	parser    *parser.Parser
	cache     hashSet
	visited   hashSet
	opts      CrawlerOptions
	result    []crawlerResult
	quit      chan os.Signal
	ticker    *time.Ticker
	ui        crawlerUi
}

type crawlerResult struct {
	URL    string   `json:"url" xml:"url,attr"`
	Status int      `json:"status" xml:"status,attr"`
	Error  string   `json:"error,omitempty" xml:"error,attr"`
	Count  int      `json:"count" xml:"linkCount,attr"`
	Links  []string `json:"links,omitempty" xml:"link"`
}

type crawlerUi struct {
	multi    *pterm.MultiPrinter
	progress *pterm.ProgressbarPrinter
	spinners []*pterm.SpinnerPrinter
}

func NewCrawler(opts CrawlerOptions) *Crawler {
	hclog.Default().Info("crawler initialised", "CrawlerOptions", structs.Map(opts))
	c := &Crawler{
		scheduler: scheduler.NewScheduler[string](scheduler.SchedulerOptions{
			MaxWorkers:  opts.MaxWorkers,
			Interactive: opts.Interactive,
		}),
		parser: parser.NewParser(parser.ParserOptions{
			Timeout:           time.Second * time.Duration(opts.RequestDeadline),
			SameSubdomain:     true,
			Distinct:          true,
			IgnoreFragments:   opts.IgnoreFragments,
			IgnoredExtensions: opts.IgnoredExtensions,
			IgnoredPaths:      opts.IgnoredPaths,
		}),
		opts: opts,
		quit: make(chan os.Signal, 1),
	}

	if opts.Interactive {
		c.ui = newUi(opts)
		c.ui.multi.Start()
	}

	signal.Notify(c.quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	c.scheduler.WithHandler(c.handler)
	return c
}

func newUi(opts CrawlerOptions) crawlerUi {
	multi := pterm.DefaultMultiPrinter.WithUpdateDelay(UpdateDuration)
	progress, err := pterm.DefaultProgressbar.WithWriter(multi.NewWriter()).Start()
	if err != nil {
		panic(err)
	}

	ui := crawlerUi{
		multi:    multi,
		progress: progress,
	}

	for i := 0; i < opts.MaxWorkers; i++ {
		spinner, err := pterm.DefaultSpinner.
			WithSequence(SpinnerSequence...).
			WithDelay(UpdateDuration).
			WithWriter(multi.NewWriter()).
			WithShowTimer(false).
			Start()
		if err != nil {
			panic(err)
		}

		ui.spinners = append(ui.spinners, spinner)
	}

	return ui
}

func (c *Crawler) Crawl(url string) {
	c.ticker = time.NewTicker(UpdateDuration)
	c.scheduler.Start()

	input, err := parser.SanitiseUrl(url)
	if err != nil {
		log.Fatal(err)
	}

	hclog.Default().Debug("crawler ready, starting", "input", input)

	c.cache.add(input)
	c.scheduler.Dispatch([]string{input})
	c.run()
}

func (c *Crawler) run() {
	defer func(c *Crawler) {
		if c.opts.Interactive {
			c.ui.multi.Stop()
		}

		c.scheduler.Stop()
		c.ticker.Stop()
		c.done()
	}(c)

	for {
		select {
		case <-c.ticker.C:
			visitedSize := c.visited.size()
			cacheSize := c.cache.size()
			if c.opts.Interactive {
				c.ui.progress.Current = visitedSize
				c.ui.progress.Total = cacheSize
			}

			if visitedSize >= cacheSize {
				c.quit <- syscall.SIGQUIT
			}
		case rs := <-c.scheduler.WorkerState:
			c.ui.spinners[rs[0].(int)].UpdateText(rs[1].(string))
		case sig := <-c.quit:
			if sig != syscall.SIGQUIT {
				os.Exit(int(sig.(syscall.Signal)))
			}
			return
		}
	}
}

func (c *Crawler) done() {
	hclog.Default().Debug("crawler finished.")
	results := c.getResultString()

	if len(c.opts.OutputFile) <= 0 {
		println(results)
	} else {
		outFile := c.opts.OutputFile
		if c.opts.OutputFormat == Output_Json && !strings.HasSuffix(outFile, ".json") {
			outFile += ".json"
		} else if c.opts.OutputFormat == Output_Xml && !strings.HasSuffix(outFile, ".xml") {
			outFile += ".xml"
		}

		writeFile(outFile, results)
		hclog.Default().Debug("wrote results to file", "filename", outFile)
	}
}

func (c *Crawler) getResultString() string {
	if c.opts.OutputFormat == Output_Json {
		b, err := json.MarshalIndent(c.result, "", "  ")
		if err != nil {
			panic(err)
		}
		return string(b)
	} else if c.opts.OutputFormat == Output_Xml {
		b, err := xml.MarshalIndent(c.result, "", "  ")
		if err != nil {
			panic(err)
		}
		return string(b)
	} else {
		var builder strings.Builder
		for _, e := range c.result {
			fmt.Fprintf(&builder, "%s\n", e.URL)
			for _, l := range e.Links {
				fmt.Fprintf(&builder, "\t%s\n", l)
			}
		}
		return builder.String()
	}
}

func writeFile(filename string, data string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	f.WriteString(data)
	err = f.Sync()
	if err != nil {
		panic(err)
	}
}

func (c *Crawler) handler(input string) {
	if c.visited.has(input) {
		return
	}

	output, err := c.parser.ParseLinks(input)
	c.visited.add(input)

	if err != nil {
		if !c.opts.Interactive {
			hclog.Default().Error(
				fmt.Sprintf("[%d/%d]", c.visited.size(), c.cache.size()),
				"status", output.Status,
				"input", input,
				"error", err,
			)
		}

		return
	}

	c.result = append(c.result, crawlerResult{
		URL:    input,
		Links:  output.Links,
		Count:  len(output.Links),
		Status: output.StatusCode,
	})

	visited := c.visited.slice()
	nonVisitedLinks := []string{}
	for _, link := range output.Links {
		if slices.Contains(visited, link) {
			continue
		}

		nonVisitedLinks = append(nonVisitedLinks, link)
	}

	c.cache.addSlice(nonVisitedLinks)
	if !c.opts.Interactive {
		hclog.Default().Debug("task complete",
			"status", output.StatusCode,
			"input", input,
			"visited", c.visited.size(),
			"total", c.cache.size(),
			"new", len(nonVisitedLinks),
		)
	}

	c.scheduler.Dispatch(output.Links)
}
