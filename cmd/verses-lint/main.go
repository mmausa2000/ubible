package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var numberedRe = regexp.MustCompile(`^\s*\d+\.\s+(.+?)\s+[—–-]\s+(.+)$`)

func main() {
	files, err := filepath.Glob("./verses/*.txt")
	if err != nil {
		fmt.Println("error: cannot read ./verses:", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Println("no .txt verse files found in ./verses")
		return
	}

	exitCode := 0
	for _, f := range files {
		file, err := os.Open(f)
		if err != nil {
			fmt.Printf("%s: open error: %v\n", f, err)
			exitCode = 1
			continue
		}
		defer file.Close()

		sc := bufio.NewScanner(file)
		lineNum := 0
		bad := 0
		for sc.Scan() {
			lineNum++
			line := sc.Text()
			if len(line) == 0 {
				continue
			}
			if numberedRe.MatchString(line) {
				continue
			}
			fmt.Printf("%s:%d: does not match 'N. <Reference> — <Text>'\n", f, lineNum)
			bad++
		}
		if err := sc.Err(); err != nil {
			fmt.Printf("%s: scan error: %v\n", f, err)
			exitCode = 1
		}
		if bad == 0 {
			fmt.Printf("%s: OK\n", f)
		}
	}
	os.Exit(exitCode)
}