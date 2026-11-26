// Command hdsort is the composition root. It wires adapters and starts the
// loopback review UI. This skeleton only prints the version.
package main

import "fmt"

func main() {
	fmt.Printf("hdsort %s\n", Version())
}
