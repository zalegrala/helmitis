package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/zalegrala/helmitis/interchange"
	"github.com/zalegrala/helmitis/stamp"
)

const version = "0.0.1-dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "stamp:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("stamp", flag.ContinueOnError)
	in := fs.String("in", "", "interchange JSON file (default: stdin)")
	jsonnet := fs.String("jsonnet", "", "run this jsonnet file and use its stdout as interchange")
	out := fs.String("out", "chart", "output chart directory")
	check := fs.Bool("check", false, "compare against on-disk chart; non-zero exit on drift, no writes")
	noValidate := fs.Bool("no-validate-output", false, "skip helm lint / kubeconform on the rendered chart")
	if err := fs.Parse(args); err != nil {
		return err
	}

	data, err := readInterchange(*in, *jsonnet)
	if err != nil {
		return err
	}

	doc, err := interchange.DecodeAndValidate(data)
	if err != nil {
		return err
	}

	files, err := stamp.Build(doc)
	if err != nil {
		return err
	}

	if *check {
		drift, diffs := stamp.Check(files, *out)
		if drift {
			return fmt.Errorf("chart drift detected:\n  %s", strings.Join(diffs, "\n  "))
		}
		fmt.Fprintln(os.Stderr, "stamp: chart is up to date")
		return nil
	}

	if err := stamp.Write(files, *out); err != nil {
		return err
	}
	if !*noValidate {
		if err := stamp.ValidateOutput(*out); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "stamp: wrote %d files to %s\n", len(files), *out)
	return nil
}

func readInterchange(in, jsonnetFile string) ([]byte, error) {
	switch {
	case jsonnetFile != "":
		cmd := exec.Command("jsonnet", jsonnetFile)
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("jsonnet %s: %v: %s", jsonnetFile, err, stderr.String())
		}
		return []byte(stdout.String()), nil
	case in != "":
		return os.ReadFile(in)
	default:
		return io.ReadAll(os.Stdin)
	}
}
