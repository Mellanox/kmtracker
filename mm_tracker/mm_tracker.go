package main

import (
	"fmt"
	"os"
)

func main() {
	var tracefile string
	var verbose bool

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Printf("%s <trace_file> [-v]\n", os.Args[0])
		return
	}

	if len(os.Args) > 2 {
		if os.Args[2] == "-v" || os.Args[2] == "verbose" {
			verbose = true
		}
	} else {
		verbose = false
	}

	tracefile = os.Args[1]
	parse_mm_entries(tracefile, verbose)
}
