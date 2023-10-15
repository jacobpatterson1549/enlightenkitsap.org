package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type config struct {
	port string
}

func (cfg *config) parseArgsAndEnv(out io.Writer, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("first argument must be program name")
	}
	programName, programArgs := args[0], args[1:]
	fs := flag.NewFlagSet(programName, flag.ExitOnError)
	fs.StringVar(&cfg.port, "port", "8000", "the port to run the site on")
	if err := fs.Parse(programArgs); err != nil {
		return fmt.Errorf("parsing program args: %w", err)
	}
	if err := cfg.parseEnvVars(fs); err != nil {
		return fmt.Errorf("setting value from environment variable: %w", err)
	}
	return nil
}

func (cfg *config) parseEnvVars(fs *flag.FlagSet) error {
	var lastErr error
	fs.VisitAll(func(f *flag.Flag) {
		upperName := strings.ToUpper(f.Name)
		name := strings.ReplaceAll(upperName, "-", "_")
		val, ok := os.LookupEnv(name)
		if !ok {
			return
		}
		if err := f.Value.Set(val); err != nil {
			lastErr = err
		}
	})
	if lastErr != nil {
		return lastErr
	}
	return nil
}
