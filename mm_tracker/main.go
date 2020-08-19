package main

import (
	"fmt"
	"os"
)

func main() {
	var tracefile string
	var verbose bool
	var kernelfile string

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Printf("%s <trace_file> [-v]\n", os.Args[0])
		return
	}
	tracefile = os.Args[1]

	if len(os.Args) > 2 {
		kernelfile = os.Args[2]
	}

	if len(os.Args) > 3 {
		if os.Args[3] == "-v" || os.Args[3] == "verbose" {
			verbose = true
		}
	} else {
		verbose = false
	}

	kmem_tracker, pfn_tracker := parse_mm_entries(tracefile, verbose)
	if kmem_tracker == nil || pfn_tracker == nil {
		fmt.Printf("Error parsing trace file %v", tracefile)
		return
	}

	if len(kernelfile) > 0 {
		ksyms, err := GetLiveKernelSymbolMap(kernelfile)
		if err != nil {
			fmt.Printf("Error parsing kernel vmlinux %v", kernelfile)
			return
		}
		ksyms.tree.BstTreePrint()
	}
}
