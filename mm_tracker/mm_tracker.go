package main

import (
	"fmt"
	"os"
)

func main() {
	var tracefile string

	if len(os.Args) != 2 {
		fmt.Println("Usage:")
		fmt.Printf("%s <trace_file>\n", os.Args[0])
		return
	}

	tracefile = os.Args[1]
	parse_mm_entries(tracefile)
}
