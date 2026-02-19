// ABOUTME: Entrypoint for the tl CLI tool.
// ABOUTME: Delegates to cli.Execute() from internal/tl package.
package main

import (
	"github.com/twilwa/tl/internal/tl"
)

func main() {
	tl.Execute()
}
