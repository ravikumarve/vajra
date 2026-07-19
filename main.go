// VAJRA — MCP Ephemeral Identity Broker
// Single Go binary entry point.
package main

import "github.com/vajra/vajra/cmd/vajra"

func main() {
	vajra.Execute()
}
