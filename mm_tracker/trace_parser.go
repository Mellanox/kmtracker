package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

type MemEntry struct {
	call_type string
	order     uint64
	pfn       uint64 /* useful only for mm */

	/* kmalloc/free friends start */
	bytes_requested uint64
	bytes_allocated uint64
	ptr             uint64
	/* kmalloc/free friends end */

	call_site   string
	line_number int
	line string
	// pointer to freed entry to link to
	freeEntry *MemEntry
}

type PfnTracker struct {
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

type KmemTracker struct {
	alloc_bytes uint64
	free_bytes  uint64
	entries     []*MemEntry
	kmemmap     map[uint64]*MemEntry /* allocated addr to entry map */
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

func get_string_value_of_key(kv string, key string) (string, error) {
	if strings.Contains(kv, key) == false {
		return "", fmt.Errorf("invalid string", kv)
	}

	parts := strings.Split(kv, "=")
	if len(parts) == 1 || len(parts[1]) == 0 {
		return "", fmt.Errorf("null value")
	}
	return parts[1], nil
}

func get_mm_order(possible_kv string) (uint64, error) {

	value, err := get_uint64_value_of_key(possible_kv, "order=", 10)
	return value, err
}

func get_mm_pfn(possible_kv string) (uint64, error) {

	value, err := get_uint64_value_of_key(possible_kv, "pfn=", 10)
	return value, err
}

func decode_pfn_order(entry *MemEntry, valid_kv []string) error {
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

func get_req_bytes(possible_kv string) (uint64, error) {

	value, err := get_uint64_value_of_key(possible_kv, "bytes_req=", 10)
	return value, err
}

func get_alloc_bytes(possible_kv string) (uint64, error) {

	value, err := get_uint64_value_of_key(possible_kv, "bytes_alloc=", 10)
	return value, err
}

func get_mem_ptr(possible_kv string) (uint64, error) {

	value, err := get_uint64_value_of_key(possible_kv, "ptr=", 16)
	return value, err
}

func decode_alloc(entry *MemEntry, valid_kv []string) error {
	mem_ptr, err := get_mem_ptr(valid_kv[2])
	if err != nil {
		return err
	}
	bytes_req, req_err := get_req_bytes(valid_kv[3])
	if req_err != nil {
		return req_err
	}
	bytes_alloc, alloc_err := get_alloc_bytes(valid_kv[4])
	if alloc_err != nil {
		return alloc_err
	}
	value, call_err := get_string_value_of_key(valid_kv[1], "call_site=")
	if call_err != nil {
		return call_err
	}
	entry.call_site = value
	entry.ptr = mem_ptr
	entry.bytes_requested = bytes_req
	entry.bytes_allocated = bytes_alloc
	return nil
}

func decode_free(entry *MemEntry, valid_kv []string) error {
	free_ptr, err := get_mem_ptr(valid_kv[2])
	if err != nil {
		return err
	}
	value, call_err := get_string_value_of_key(valid_kv[1], "call_site=")
	if call_err != nil {
		return call_err
	}
	entry.call_site = value
	entry.ptr = free_ptr
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
	memEntry.line = line
	memEntry.call_type = strings.TrimRight(valid_kv[0], ":")
	//fmt.Printf("%s ", memEntry.call_type)

	switch memEntry.call_type {
	case "mm_page_alloc", "mm_page_alloc_zone_locked", "mm_page_free", "mm_page_free_batched":
		err := decode_pfn_order(memEntry, valid_kv)
		if err != nil {
			return nil, err
		}
	case "kmem_cache_alloc", "kmalloc_node", "kmalloc":
		err := decode_alloc(memEntry, valid_kv)
		if err != nil {
			return nil, err
		}
	case "kmem_cache_free", "kfree":
		err := decode_free(memEntry, valid_kv)
		if err != nil {
			return nil, err
		}
	}
	return memEntry, nil
}

func order_to_bytes(order uint64) uint64 {
	return (order + 1) * 4096
}

func parse_mm_entries(trace_file string, verbose bool) {
	pfn_tracker := new(PfnTracker)
	pfn_tracker.pfnmap = make(map[uint64]*MemEntry)

	kmem_tracker := new(KmemTracker)
	kmem_tracker.kmemmap = make(map[uint64]*MemEntry)

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
		case "mm_page_alloc", "mm_page_alloc_zone_locked":
			pfn_tracker.entries = append(pfn_tracker.entries, entry)
			pfn_tracker.pfnmap[entry.pfn] = entry
			pfn_tracker.alloc_bytes += order_to_bytes(entry.order)
		case "mm_page_free", "mm_page_free_batched":
			pfn_tracker.entries = append(pfn_tracker.entries, entry)
			if pfn_tracker.pfnmap[entry.pfn] != nil {
				pfn_tracker.free_bytes += order_to_bytes(entry.order)
				pfn_tracker.pfnmap[entry.pfn] = nil
			}
		case "kmem_cache_alloc", "kmalloc_node", "kmalloc":
			kmem_tracker.entries = append(kmem_tracker.entries, entry)
			kmem_tracker.alloc_bytes += entry.bytes_allocated
			kmem_tracker.kmemmap[entry.ptr] = entry
		case "kmem_cache_free", "kfree":
			kmem_tracker.entries = append(kmem_tracker.entries, entry)
			if kmem_tracker.kmemmap[entry.ptr] != nil {
				kmem_tracker.free_bytes += kmem_tracker.kmemmap[entry.ptr].bytes_allocated
				kmem_tracker.kmemmap[entry.ptr] = nil
			}
		}
	}
	fmt.Printf("page alloc bytes = %d = %d Kbytes = %d Mbytes\n", pfn_tracker.alloc_bytes,
		pfn_tracker.alloc_bytes/1024, pfn_tracker.alloc_bytes/(1024*1024))
	fmt.Printf("page free bytes = %d = %d Kbytes = %d Mbytes\n", pfn_tracker.free_bytes,
		pfn_tracker.free_bytes/1024, pfn_tracker.free_bytes/(1024*1024))
	fmt.Printf("kmem alloc bytes = %d = %d Kbytes = %d Mbytes\n", kmem_tracker.alloc_bytes,
		kmem_tracker.alloc_bytes/1024, kmem_tracker.alloc_bytes/(1024*1024))
	fmt.Printf("kmem free bytes = %d = %d Kbytes = %d Mbytes\n", kmem_tracker.free_bytes,
		kmem_tracker.free_bytes/1024, kmem_tracker.free_bytes/(1024*1024))

	if verbose {
		for _, element := range kmem_tracker.kmemmap {
			if element == nil {
				continue
			}
			fmt.Printf("%v\n", element.line)
		}
	}
}
