package main

import (
	"fmt"
	"os"
	"strconv"
)

func MapTraceToSymbol(tracker *MemEntryTracker, symbols *KernelSymbols) {
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
	}
}

func printTrackerSummary(tracker *MemEntryTracker) {
	fmt.Printf("%v done for size = %v, for %v times\n",
		   tracker.name, tracker.size, tracker.count)
}

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
		if found == true {
			fmt.Println(tracker.name, mementry.call_site_fn,
				    mementry.length, mementry.index)
		}
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
		fmt.Printf("meme trneie")
		return
	}

	newmap, err := GetLiveKernelSymbolMap(kernelfile)
	if err != nil {
		fmt.Printf("kernel")
		return
	}

	fmt.Println("Mapping trace entries with symbols, please wait...")
	MapTraceToSymbol(&memEntries.kmalloc, newmap)
	MapTraceToSymbol(&memEntries.kmalloc_node, newmap)
	MapTraceToSymbol(&memEntries.kmem_cache_alloc, newmap)
	MapTraceToSymbol(&memEntries.kfree, newmap)
	MapTraceToSymbol(&memEntries.kmem_cache_free, newmap)

	if len(os.Args) > 4 && (os.Args[4] == "verbose" || os.Args[4] == "v" ||
		os.Args[4] == "-v") {
		printForTracker(&memEntries.kmalloc, newmap)
		printForTracker(&memEntries.kmalloc_node, newmap)
		printForTracker(&memEntries.kmem_cache_alloc, newmap)
		printForTracker(&memEntries.kfree, newmap)
		printForTracker(&memEntries.kmem_cache_free, newmap)
	}

	printTrackerSummary(&memEntries.kmalloc)
	printTrackerSummary(&memEntries.kmalloc_node)
	printTrackerSummary(&memEntries.kmem_cache_alloc)
	printTrackerSummary(&memEntries.kfree)
	printTrackerSummary(&memEntries.kmem_cache_free)
}
