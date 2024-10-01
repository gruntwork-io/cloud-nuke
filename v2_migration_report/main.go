//go:generate go run main.go
package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/gruntwork-io/cloud-nuke/aws"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("failed to get current working directory")
	}

	file, err := os.Create(cwd + "/output.md")
	if err != nil {
		log.Fatal("failed to create output file")
	}
	defer file.Close()

	_, _ = fmt.Fprintln(file, "# AWS SDKv2 Migration Progress")
	_, _ = fmt.Fprintln(file)
	_, _ = fmt.Fprintln(file, "The table below outlines the progress of the `AWS SDK` migration as detailed in [#745](https://github.com/gruntwork-io/cloud-nuke/issues/745).")
	_, _ = fmt.Fprintln(file, "run `go generate ./...` to refresh this report.")
	_, _ = fmt.Fprintln(file)
	_, _ = fmt.Fprintln(file)

	var (
		resources       = aws.ReportGetAllRegisterResources()
		migrationStatus = make(map[string]string)
	)
	const checked = ":white_check_mark:"

	for _, item := range resources {
		status := ""
		if item.IsUsingV2() {
			status = checked
		}

		migrationStatus[item.ResourceName()] = status
	}

	var (
		keys      []string
		keyMaxLen int
	)
	for k := range migrationStatus {
		keys = append(keys, k)
		if len(k) > keyMaxLen {
			keyMaxLen = len(k)
		}
	}
	sort.Strings(keys)

	w := tabwriter.NewWriter(file, 0, 0, 0, ' ', 0)
	_, _ = fmt.Fprintln(w, "| Resource Name\t| Migrated\t|")
	_, _ = fmt.Fprintf(w, "|%s\t|%s\t|\n", dash(keyMaxLen), dash(len(checked)))

	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "| %s\t| %s\t|\n", k, migrationStatus[k])
	}

	_ = w.Flush()
}

func dash(n int) string {
	return strings.Repeat("-", n+2)
}
