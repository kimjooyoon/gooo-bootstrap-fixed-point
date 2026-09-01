package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-bootstrap-fixed-point/internal/fixedpoint"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "conformance":
		if err := runConformance(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "integration":
		if err := runIntegration(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "generate":
		if err := runGenerate(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func runConformance(args []string) error {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	input := flags.String("input", "examples/self-description.gooo", "Gooo semantic source")
	root := flags.String("root", ".", "repository root containing contracts and fixtures")
	output := flags.String("output", "", "caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *output == "" {
		return fmt.Errorf("--output is required")
	}
	_, err := fixedpoint.RunConformance(*input, *root, *output)
	return err
}

func runIntegration(args []string) error {
	flags := flag.NewFlagSet("integration", flag.ContinueOnError)
	input := flags.String("input", "examples/self-description.gooo", "Gooo semantic source")
	output := flags.String("output", "", "caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *output == "" {
		return fmt.Errorf("--output is required")
	}
	return fixedpoint.RunIntegration(*input, *output)
}

func runGenerate(args []string) error {
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	input := flags.String("input", "examples/self-description.gooo", "Gooo semantic source")
	stage := flags.String("stage", "G1", "generation stage")
	output := flags.String("output", "", "caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *output == "" {
		return fmt.Errorf("--output is required")
	}
	raw, err := os.ReadFile(*input)
	if err != nil {
		return err
	}
	spec, err := fixedpoint.ParseSource(raw)
	if err != nil {
		return err
	}
	if *stage != "G0" && *stage != "G1" && *stage != "G2" {
		return fmt.Errorf("stage must be G0, G1, or G2")
	}
	return fixedpoint.WriteGenerated(spec, raw, *stage, filepath.Clean(*output))
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gooo-bootstrap {conformance|integration|generate} [flags]")
}
