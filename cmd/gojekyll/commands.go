package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/urfave/cli.v1"
	yaml "gopkg.in/yaml.v2"

	"github.com/osteele/gojekyll"
	"github.com/osteele/gojekyll/helpers"
)

// main sets this
var commandStartTime = time.Now()

func buildCommand(c *cli.Context, site *gojekyll.Site) error {
	printPathSetting("Destination:", site.Destination)
	printSetting("Generating...", "")
	if buildOptions.DryRun {
		buildOptions.Verbose = true
	}
	count, err := site.Build(buildOptions)
	if err != nil {
		return err
	}
	elapsed := time.Since(commandStartTime)
	printSetting("", fmt.Sprintf("created %d files in %.2fs.", count, elapsed.Seconds()))
	return nil
}

func benchmarkCommand(c *cli.Context, site *gojekyll.Site) error {
	printSetting("Generating...", "")
	for i := 0; i < 10; i++ {
		printSetting("", fmt.Sprintf("%d", i+1))
		_, err := site.Build(buildOptions)
		if err != nil {
			return err
		}
	}
	return nil
	// elapsed := time.Since(commandStartTime)
	// printSetting("", fmt.Sprintf("created %d files in %.2fs.", count, elapsed.Seconds()))
}

func serveCommand(c *cli.Context, site *gojekyll.Site) error {
	server := gojekyll.Server{Site: site}
	return server.Run(printSetting)
}

func dataCommand(c *cli.Context, site *gojekyll.Site) error {
	p, err := cliPage(c, site)
	if err != nil {
		return err
	}

	printSetting("Variables:", "")
	// The YAML representation including collections is impractically large for debugging.
	// (Actually it's circular, which the yaml package can't handle.)
	// Neuter it. This destroys it as Liquid data, but that's okay in this context.
	siteData := site.Variables
	for _, c := range site.Collections {
		siteData[c.Name] = fmt.Sprintf("<elided page data for %d items>", len(siteData[c.Name].([]gojekyll.VariableMap)))
	}
	b, _ := yaml.Marshal(p.DebugVariables())
	fmt.Println(string(b))
	return nil
}

func routesCommand(c *cli.Context, site *gojekyll.Site) error {
	printSetting("Routes:", "")
	urls := []string{}
	for u, p := range site.Paths {
		if !(c.Bool("dynamic") && p.Static()) {
			urls = append(urls, u)
		}
	}
	sort.Strings(urls)
	for _, u := range urls {
		fmt.Printf("  %s -> %s\n", u, site.Paths[u].Path())
	}
	return nil
}

func renderCommand(c *cli.Context, site *gojekyll.Site) error {
	page, err := cliPage(c, site)
	if err != nil {
		return err
	}
	printPathSetting("Render:", filepath.Join(site.Source, page.Path()))
	printSetting("URL:", page.Permalink())
	printSetting("Content:", "")
	return page.Write(os.Stdout)
}

// If path starts with /, it's a URL path. Else it's a file path relative
// to the site source directory.
func cliPage(c *cli.Context, site *gojekyll.Site) (page gojekyll.Page, err error) {
	arg := "/"
	if c.NArg() > 0 {
		arg = c.Args().Get(0)
	}
	if strings.HasPrefix(arg, "/") {
		page, _ = site.PageForURL(arg)
		if page == nil {
			err = helpers.NewPathError("render", arg, "the site does not include a file with this URL path")
		}
	} else {
		page = site.FindPageByFilePath(arg)
		if page == nil {
			err = helpers.NewPathError("render", arg, "no such file")
		}
	}
	return
}

// Load the site specified at destination into the site global, and print the common banner settings.
func loadSite(source, destination string) (*gojekyll.Site, error) {
	site, err := gojekyll.NewSiteFromDirectory(source)
	if err != nil {
		return nil, err
	}
	if destination != "" {
		site.Destination = destination
	}
	if site.ConfigFile != nil {
		printPathSetting(configurationFileLabel, *site.ConfigFile)
	} else {
		printSetting(configurationFileLabel, "none")

	}
	printPathSetting("Source:", site.Source)
	return site, site.ReadFiles()
}

// Given a subcommand function, load the site and then call the subcommand.
func loadSiteAndRun(siteLoader func() (*gojekyll.Site, error), cmd func(*cli.Context, *gojekyll.Site) error) func(*cli.Context) error {
	return func(c *cli.Context) error {
		site, err := siteLoader()
		if err != nil {
			return cli.NewExitError(err, 1)
		}
		if err := cmd(c, site); err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	}
}