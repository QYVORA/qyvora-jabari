// Command jabari is the entry point for the JABARI Android Security
// Assessment Framework.
//
// The same binary is also published under the alias "androidsec" so both
// command names work in scripts and documentation.
package main

import (
	"os"

	"github.com/QYVORA/qyvora-jabari/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
