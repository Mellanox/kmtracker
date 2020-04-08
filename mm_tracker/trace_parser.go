package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

type MemEntry struct {
	call_type   string
	order       uint64
	pfn         uint64
	line_number int
	// pointer to freed entry to link to
	freeEntry *MemEntry
}

type PfnTracker struct {
	name        string
	alloc_bytes uint64
	free_bytes  uint64
	entries     []*MemEntry
	pfnmap      map[uint64]*MemEntry
	/* if alloc-free is balanced, moved to balanced map
	 * so that when pfn is reallocated its free can be tracked
	 * again.
	 */
	pfnmap_balanced map[uint64][]*MemEntry
}

func get_uint64_value_of_key(kv string, key string, base int) (uint64, error) {
	if strings.Contains(kv, key) == false {
		return 0, fmt.Errorf("invalid string", kv)
	}

	parts := strings.Split(kv, "=")
	if len(parts) == 1 || len(parts[1]) == 0 {
		return 0, fmt.Errorf("null value")
	}

	value, err2 := strconv.ParseUint(parts[1], base, 64)
	if err2 != nil {
		return 0, err2
	}
	return value, nil
}

func get_mm_order(possible_kv string) (uint64, error) {

	value, err := get_uint64_value_of_key(possible_kv, "order=", 10)
	return value, err
}

func get_mm_pfn(possible_kv string) (uint64, error) {

	value, err := get_uint64_value_of_key(possible_kv, "pfn=", 10)
	return value, err
}

func decode_pfn_order(entry *MemEntry, valid_kv []string) (error) {
	//fmt.Printf("%s %s\n", valid_kv[2], valid_kv[3])
	pfn, pfn_err := get_mm_pfn(valid_kv[2])
	if pfn_err != nil {
		return pfn_err
	}
	order, order_err := get_mm_order(valid_kv[3])
	if order_err != nil {
		return order_err
	}
	entry.pfn = pfn
	entry.order = order
	return nil
}

func parseLine(line string, line_number int) (*MemEntry, error) {

	line = strings.TrimLeft(line, " ")
	if len(line) == 0 {
		return nil, fmt.Errorf("short line")
	}

	words := strings.Split(line, " ")

	valid_kv := make([]string, 0)

	for _, word := range words {
		if len(word) == 0 || word == " " || word == "(null)" {
			continue
		}
		//fmt.Println(valid_kv)
		valid_kv = append(valid_kv, word)
	}
	if len(valid_kv) < 4 {
		return nil, fmt.Errorf("short line")
	}

	//remove pid, cpu, state, time
	valid_kv = valid_kv[4:]
	//fmt.Println(pid, len(valid_kv), valid_kv)

	memEntry := new(MemEntry)
	memEntry.line_number = line_number
	memEntry.call_type = strings.TrimRight(valid_kv[0], ":")
	//fmt.Printf("%s ", memEntry.call_type)

	switch memEntry.call_type {
	case "mm_page_alloc":
		err := decode_pfn_order(memEntry, valid_kv)
		if err != nil {
			return nil, err
		}
	case "mm_page_alloc_zone_locked":
		err := decode_pfn_order(memEntry, valid_kv)
		if err != nil {
			return nil, err
		}
	case "mm_page_free":
		err := decode_pfn_order(memEntry, valid_kv)
		if err != nil {
			return nil, err
		}
	case "mm_page_free_batched":
		err := decode_pfn_order(memEntry, valid_kv)
		if err != nil {
			return nil, err
		}
	}
	return memEntry, nil
}

func order_to_bytes(order uint64) (uint64) {
	return (order + 1) * 4096
}

func parse_mm_entries(trace_file string) {
	pfn_tracker := new(PfnTracker)
	pfn_tracker.pfnmap = make(map[uint64]*MemEntry)

	data, err := ioutil.ReadFile(trace_file)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	//skip first 11 lines
	offset := 11
	lines = lines[offset:]

	for i, line := range lines {
		entry, err := parseLine(line, i+offset+1)
		if err != nil {
			continue
		}

		switch entry.call_type {
		case "mm_page_alloc":
			pfn_tracker.entries = append(pfn_tracker.entries, entry)
			pfn_tracker.pfnmap[entry.pfn] = entry
			pfn_tracker.alloc_bytes += order_to_bytes(entry.order)
		case "mm_page_alloc_zone_locked":
			pfn_tracker.entries = append(pfn_tracker.entries, entry)
			pfn_tracker.pfnmap[entry.pfn] = entry
			pfn_tracker.alloc_bytes += order_to_bytes(entry.order)
		case "mm_page_free":
			pfn_tracker.entries = append(pfn_tracker.entries, entry)
			pfn_tracker.free_bytes += order_to_bytes(entry.order)
		case "mm_page_free_batched":
			pfn_tracker.entries = append(pfn_tracker.entries, entry)
			pfn_tracker.free_bytes += order_to_bytes(entry.order)
		}
	}
	fmt.Printf("alloc bytes = %d = %d Kbytes = %d Mbytes\n", pfn_tracker.alloc_bytes,
		  pfn_tracker.alloc_bytes / 1024, pfn_tracker.alloc_bytes / (1024 * 1024))
	fmt.Printf("free bytes = %d = %d Kbytes = %d Mbytes\n", pfn_tracker.free_bytes,
		  pfn_tracker.free_bytes / 1024, pfn_tracker.free_bytes / (1024 * 1024))
}
