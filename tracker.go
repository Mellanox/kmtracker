package main

import (
	"fmt"
	"os"
	"strconv"
)

func printForTracker(tracker *MemEntryTracker, symbols *KernelSymbols) {
	var found bool

	for _, mementry := range tracker.entries {
		found = false
		for k, _ := range symbols.ModulesSymbols {
			for _, sym := range symbols.ModulesSymbols[k].Symbols {
				if mementry.call_site >= sym.StartAddress &&
					mementry.call_site <= sym.EndAddress {
					mementry.call_site_fn = sym.Name
					found = true
					break
				}
			}
			if found == true {
				break
			}
		}
		fmt.Println(tracker.name, mementry.call_site_fn, mementry.length, mementry.index)
	}
}

func main() {
	var kernelfile string
	var tracefile string
	var err error
	var pid int

	if len(os.Args) != 4 {
		fmt.Println("Usage:")
		fmt.Printf("%s trace_file pid_to_analyse vmlinux_file\n", os.Args[0])
		return
	}

	if len(os.Args) > 2 {
		pid, err = strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("invalid pid")
			return
		}
	}

	if len(os.Args) > 3 {
		kernelfile = os.Args[3]
	}

	tracefile = os.Args[1]
	memEntries, errm := BuildMemEntries(tracefile, int32(pid))
	if errm != nil {
		return
	}

	newmap, err := GetLiveKernelSymbolMap(kernelfile)
	if err != nil {
		return
	}

	printForTracker(&memEntries.kmalloc, newmap)
	printForTracker(&memEntries.kmalloc_node, newmap)
	printForTracker(&memEntries.kmem_cache_alloc, newmap)
	printForTracker(&memEntries.kfree, newmap)
	printForTracker(&memEntries.kmem_cache_free, newmap)
}
