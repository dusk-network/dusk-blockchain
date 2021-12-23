// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//go:embed indexheader.md
var indexHeader string

var root = "."

func main() {
	target, err := os.OpenFile("./index.md", os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Fprintln(target, indexHeader)

	err = filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {

			if strings.HasSuffix(path, "README.md") {

				fmt.Fprintf(target, "### [%s](./%s)\n\n", path, path)

				src, err := ioutil.ReadFile(path)
				if err != nil {
					return nil
				}

				split := strings.Split(string(src), "<!-- ToC start -->")
				sc := bufio.NewScanner(strings.NewReader(split[0]))
				var lines []string

				for sc.Scan() {
					txt := sc.Text()
					// skip the header and blank lines
					if !strings.HasPrefix(txt, "#") {
						lines = append(lines, sc.Text())
					}
				}

				trimmed := strings.TrimSpace(strings.Join(lines, "\n"))

				fmt.Fprintln(target, trimmed)
			}

			return nil
		},
	)

	if err != nil {
		fmt.Println("error:", err)
	}
}
