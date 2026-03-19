package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var readmePath string

func init() {
	docsCmd.Flags().StringVarP(&readmePath, "path", "p", "README.md", "Path to the readme file")
	rootCmd.AddCommand(docsCmd)
}

var docsCmd = &cobra.Command{
	Use:    "docs",
	Short:  "Generate documentation for vpngate",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		content, err := os.ReadFile(readmePath)
		if err != nil {
			return err
		}

		startMarker := "<!-- cobra:docs:start -->"
		endMarker := "<!-- cobra:docs:end -->"

		startIdx := bytes.Index(content, []byte(startMarker))
		endIdx := bytes.Index(content, []byte(endMarker))

		if startIdx == -1 || endIdx == -1 {
			return fmt.Errorf("markers not found in %s", readmePath)
		}

		var buf bytes.Buffer
		linkHandler := func(name string) string {
			base := strings.TrimSuffix(name, filepath.Ext(name))
			return "#" + strings.ReplaceAll(base, "_", "-")
		}

		// Disable the auto generation tag
		rootCmd.DisableAutoGenTag = true

		if err := genMarkdownRecursive(rootCmd, &buf, linkHandler); err != nil {
			return err
		}

		indented := postProcess(buf.String())

		newContent := append(content[:startIdx+len(startMarker)], []byte("\n")...)
		newContent = append(newContent, []byte(indented)...)
		newContent = append(newContent, content[endIdx:]...)

		if err := os.WriteFile(readmePath, newContent, 0644); err != nil {
			return err
		}
		fmt.Printf("Updated %s\n", readmePath)

		return nil
	},
}

// postProcess increases every markdown heading level by one (only outside
// fenced code blocks) so that the generated docs nest naturally under the
// existing ## Usage heading. It also strips the repetitive "SEE ALSO"
// sections that Cobra generates.
func postProcess(s string) string {
	var out strings.Builder
	lines := strings.Split(s, "\n")
	inFence := false
	inSeeAlso := false
	seeAlsoLevel := ""

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Track fenced code blocks so we don't mangle their contents.
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
		}

		if !inFence {
			// Detect a "SEE ALSO" heading at any level (e.g. "### SEE ALSO").
			trimmed := strings.TrimLeft(line, "#")
			if len(trimmed) != len(line) && strings.TrimSpace(trimmed) == "SEE ALSO" {
				inSeeAlso = true
				seeAlsoLevel = line[:len(line)-len(trimmed)] // heading prefix
				continue
			}

			// While inside a SEE ALSO section, skip lines until we hit the
			// next heading at the same or higher level (fewer or equal #'s).
			if inSeeAlso {
				if strings.HasPrefix(line, "#") {
					level := line[:len(line)-len(strings.TrimLeft(line, "#"))]
					if len(level) <= len(seeAlsoLevel) {
						inSeeAlso = false
						// Fall through to process this line normally.
					} else {
						continue
					}
				} else {
					continue
				}
			}

			// Indent headings by one level.
			if strings.HasPrefix(line, "#") {
				out.WriteString("#")
			}
		}

		out.WriteString(line)
		out.WriteString("\n")
	}

	return out.String()
}

func genMarkdownRecursive(cmd *cobra.Command, w io.Writer, linkHandler func(string) string) error {
	if err := doc.GenMarkdownCustom(cmd, w, linkHandler); err != nil {
		return err
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := genMarkdownRecursive(c, w, linkHandler); err != nil {
			return err
		}
	}
	return nil
}
