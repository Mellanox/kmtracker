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
	fmt.Printf("Total %v size = %v bytes, called %v times\n",
		tracker.name, tracker.size, tracker.count)
}

func printPairs(tracker *MemEntryTracker) {
	var freeTracker string

	if len(tracker.entries) != 0 {
		fmt.Println("alloc_type, caller function, allocated_length(bytes), index_in_file")
	}
	switch tracker.name {
	case "kmalloc":
		freeTracker = "kfree"
	case "kmalloc_node":
		freeTracker = "kfree"
	case "kmem_cache_alloc":
		freeTracker = "kmem_cache_free"
	default:
		freeTracker = "unknown"
	}

	for _, mementry := range tracker.entries {
		if len(mementry.call_site_fn) != 0 && mementry.freeEntry != nil {
			fmt.Println(tracker.name, mementry.call_site_fn,
				mementry.length, mementry.index)
			fmt.Printf("%v %v %v %v\n", freeTracker, mementry.freeEntry.call_site_fn,
				mementry.freeEntry.length, mementry.freeEntry.index)
		}
	}
}

func printLoners(tracker *MemEntryTracker) {
	if len(tracker.entries) != 0 {
		fmt.Println("alloc_type, caller function, allocated_length(bytes), index_in_file")
	}
	for _, mementry := range tracker.entries {
		if len(mementry.call_site_fn) != 0 && mementry.freeEntry == nil {
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

	if len(os.Args) != 4 && len(os.Args) != 5 {
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
		printPairs(&memEntries.kmalloc)
		printPairs(&memEntries.kmalloc_node)
		printPairs(&memEntries.kmem_cache_alloc)
		printLoners(&memEntries.kmalloc)
		printLoners(&memEntries.kmalloc_node)
		printLoners(&memEntries.kmem_cache_alloc)
		printLoners(&memEntries.kfree)
		printLoners(&memEntries.kmem_cache_free)
	}
	printTrackerSummary(&memEntries.kmalloc)
	printTrackerSummary(&memEntries.kmalloc_node)
	printTrackerSummary(&memEntries.kmem_cache_alloc)
	printTrackerSummary(&memEntries.kfree)
	printTrackerSummary(&memEntries.kmem_cache_free)
	printTrackerSummary(&memEntries.mm_page_alloc)
	printTrackerSummary(&memEntries.mm_page_free)
	fmt.Println("-------------------------------------------------")
	fmt.Printf("Total alloc size = %v bytes\n", memEntries.allocSize)
	fmt.Printf("Total free size = %v bytes\n", memEntries.freeSize)
	fmt.Printf("Total page memory alloc size = %v bytes\n", memEntries.pageAllocSize)
	fmt.Printf("Total page memory free size = %v bytes\n", memEntries.pageFreeSize)
	fmt.Printf("Total kernel memory allocated = %v bytes\n", memEntries.allocSize-memEntries.freeSize)
}
