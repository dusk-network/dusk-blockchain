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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("file path of the repository is required")
		os.Exit(1)
	}
	target, err := os.OpenFile("./index.md", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(target, indexHeader)
	root := os.Args[1]
	err = filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			
			if err != nil {
				fmt.Println(err)
				return nil
			}
			if strings.HasSuffix(path, "README.md") {
				fmt.Fprintf(target, "### [%s](./%s)\n\n", path, path)
				src, err := ioutil.ReadFile(path)
				if err != nil {
					return nil
				}
				split := strings.Split(string(src), "<!-- ToC start -->")
				var lines []string
				sc := bufio.NewScanner(strings.NewReader(split[0]))
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
