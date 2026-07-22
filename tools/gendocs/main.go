// Command gendocs regenerates docs/cli/ from vpngate's actual cobra command
// tree (Use/Short/Long/flags), via cobra's own doc.GenMarkdownTree — a
// mechanical flag/usage reference, not a replacement for README.md's
// narrative sections. Run via `make docs`; not part of the shipped vpngate
// binary, so it lives outside cmd/vpngate.
package main

import (
	"log"
	"os"

	"github.com/davegallant/vpngate/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	const dir = "docs/cli"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("creating %s: %v", dir, err)
	}
	if err := doc.GenMarkdownTree(cmd.RootCmd(), dir); err != nil {
		log.Fatalf("generating docs: %v", err)
	}
}
