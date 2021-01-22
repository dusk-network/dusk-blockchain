// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// GOLANGCI_VERSION to be used for linting.
	GOLANGCI_VERSION = "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.31.0"
)

// GOBIN environment variable.
func GOBIN() string {
	if os.Getenv("GOBIN") == "" {
		log.Fatal("GOBIN not set")
	}

	return os.Getenv("GOBIN")
}

func main() {
	log.SetFlags(log.Lshortfile)

	if _, err := os.Stat(filepath.Join("scripts", "build.go")); os.IsNotExist(err) {
		log.Fatal("should run build from root dir")
	}

	if len(os.Args) < 2 {
		log.Fatal("cmd required, eg: install")
	}

	switch os.Args[1] {
	case "install":
		install(os.Args[2:])
	case "lint":
		lint()
	default:
		log.Fatal("cmd not found: ", os.Args[1])
	}
}

//nolint:gosec
func install(cmdline []string) {
	var (
		race  = flag.Bool("race", false, "build Dusk exec with Race enabled")
		debug = flag.Bool("debug", false, "build Dusk exec with Debug enabled")
	)

	_ = flag.CommandLine.Parse(cmdline)

	argsList := append([]string{"list"}, []string{"./..."}...)

	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), argsList...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("could not list packages: %v\n%s", err, string(out))
	}

	var packages []string

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "/dusk-blockchain/") && !strings.Contains(line, "/harness") {
			packages = append(packages, strings.TrimSpace(line))
		}
	}

	argsInstall := []string{"install"}
	if *race {
		argsInstall = append(argsInstall, "-race")
	}

	if *debug {
		argsInstall = append(argsInstall, "-gcflags=\"all=-N -l\"")
	}

	cmd = exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), argsInstall...)
	cmd.Args = append(cmd.Args, "-v")
	cmd.Args = append(cmd.Args, packages...)

	fmt.Println("Building Dusk ...", strings.Join(cmd.Args, " \\\n"))
	cmd.Stderr, cmd.Stdout = os.Stderr, os.Stdout

	if err := cmd.Run(); err != nil {
		log.Fatal("Error: Could not build Dusk. ", "error: ", err, ", cmd: ", cmd)
	}
}

//nolint:gosec
func lint() {
	v := flag.Bool("v", false, "log verbosely")

	// Make sure GOLANGCI is downloaded and available.
	argsGet := []string{"get", GOLANGCI_VERSION}
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), argsGet...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("could not list pkgs: %v\n%s", err, string(out))
	}

	cmd = exec.Command(filepath.Join(GOBIN(), "golangci-lint"))
	cmd.Args = append(cmd.Args, "run", "--config", ".golangci.yml")

	if *v {
		cmd.Args = append(cmd.Args, "-v")
	}

	fmt.Println("Linting Dusk ...", strings.Join(cmd.Args, " \\\n"))
	cmd.Stderr, cmd.Stdout = os.Stderr, os.Stdout

	if err := cmd.Run(); err != nil {
		log.Fatal("Error: Could not Lint Dusk ... ", "error: ", err, ", cmd: ", cmd)
	}
}
