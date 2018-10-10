package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MemEntry struct {
	pid          int32
	call_type    string
	call_site    uint64
	call_site_fn string
	ptr          uint64
	length       uint64

	// index in file so that we can compare
	// free with malloc with lower index than
	// the one found for free.
	index int

	// pointer to freed entry to link to
	freeEntry *MemEntry
}

type MemEntryTracker struct {
	size    uint64
	count   uint64
	entries []*MemEntry
	ptr_map map[uint64][]*MemEntry
	name    string
}

type MemEntrieByType struct {
	kmalloc      MemEntryTracker
	kmalloc_node MemEntryTracker
	kfree        MemEntryTracker

	kmem_cache_alloc MemEntryTracker
	kmem_cache_free  MemEntryTracker

	mm_page_alloc MemEntryTracker
	mm_page_free  MemEntryTracker

	allocSize uint64
	freeSize  uint64

	pageAllocSize uint64
	pageFreeSize  uint64
}

func getpid(task_pid string) int32 {

	parts := strings.Split(task_pid, "-")
	if len(parts) != 2 {
		return -1
	}
	pid, err := strconv.ParseInt(parts[1], 0, 32)
	if err != nil {
		return -1
	}
	return int32(pid)
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

func get_call_site_addr(call_site string) (uint64, error) {

	value, err := get_uint64_value_of_key(call_site, "call_site=", 16)
	return value, err
}

func get_ptr_value(ptr string) (uint64, error) {

	value, err := get_uint64_value_of_key(ptr, "ptr=", 16)
	return value, err
}

func get_kmalloc_bytes(possible_kv string) (uint64, error) {

	value, err := get_uint64_value_of_key(possible_kv, "bytes_alloc=", 10)
	return value, err
}

func get_kmalloc_bytes_from_line(words []string) (uint64, error) {

	for _, word := range words {
		value, err := get_kmalloc_bytes(word)
		if err == nil {
			return value, nil
		}
	}
	return 0, fmt.Errorf("not found")
}

func get_mm_alloc_bytes(possible_kv string) (uint64, error) {

	value, err := get_uint64_value_of_key(possible_kv, "order=", 10)
	return value, err
}

func get_mm_alloc_bytes_from_line(words []string) (uint64, error) {

	for _, word := range words {
		value, err := get_mm_alloc_bytes(word)
		if err == nil {
			return (value + 1) * uint64(os.Getpagesize()), nil
		}
	}
	return 0, fmt.Errorf("not found")
}

func parseLine(line string, index int, search_pid int32) (*MemEntry, error) {

	line = strings.TrimLeft(line, " ")
	if len(line) == 0 {
		return nil, fmt.Errorf("short line")
	}

	words := strings.Split(line, " ")
	pid := getpid(words[0])

	if pid != search_pid {
		return nil, fmt.Errorf("pid mismatch")
	}

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

	addr, _ := get_call_site_addr(valid_kv[1])
	ptr, _ := get_ptr_value(valid_kv[2])
	memEntry := new(MemEntry)
	memEntry.pid = pid
	memEntry.call_type = strings.TrimRight(valid_kv[0], ":")
	memEntry.call_site = addr
	memEntry.ptr = ptr
	memEntry.index = index
	/*
		if ptr != 0 {
			fmt.Printf("type = %v ptr = 0x%x\n ", memEntry.call_type, ptr)
		}
	*/

	if memEntry.call_type == "kmalloc" ||
		memEntry.call_type == "kmalloc_node" ||
		memEntry.call_type == "kmem_cache_alloc" {
		length, len_err := get_kmalloc_bytes_from_line(valid_kv)
		if len_err != nil {
			return nil, len_err
		}
		memEntry.length = length
	} else if memEntry.call_type == "mm_page_alloc" {
		length, len_err := get_mm_alloc_bytes_from_line(valid_kv)
		if len_err != nil {
			return nil, len_err
		}
		memEntry.length = length
	} else if memEntry.call_type == "mm_page_free" {
		length, len_err := get_mm_alloc_bytes_from_line(valid_kv)
		if len_err != nil {
			return nil, len_err
		}
		memEntry.length = length
	}
	return memEntry, nil
}

func IntMemTracker(tracker *MemEntryTracker, name string) {
	tracker.name = name
	tracker.ptr_map = make(map[uint64][]*MemEntry)
}

func LinkFreeEntryToAlloc(memEntries *MemEntrieByType,
	mallocTracker *MemEntryTracker, freeEntry *MemEntry) uint64 {
	var malloc_len uint64
	var i int

	if len(mallocTracker.ptr_map[freeEntry.ptr]) == 0 {
		return 0
	}

	for i = 0; i < len(mallocTracker.ptr_map[freeEntry.ptr]); i++ {
		malloc_len = mallocTracker.ptr_map[freeEntry.ptr][i].length
		mallocTracker.ptr_map[freeEntry.ptr][i].freeEntry = freeEntry
		break
	}

	return malloc_len
}

func linkFreeToAlloc(memEntries *MemEntrieByType) {

	var freeTracker *MemEntryTracker

	freeTracker = &memEntries.kfree
	for _, freeentry := range freeTracker.entries {
		if freeentry.ptr == 0 {
			//kfree with zero is valid in kernel
			continue
		}
		len := LinkFreeEntryToAlloc(memEntries, &memEntries.kmalloc, freeentry)
		if len == 0 {
			len = LinkFreeEntryToAlloc(memEntries,
				&memEntries.kmalloc_node, freeentry)
		} else {
			freeentry.length = len
		}
		if len != 0 {
			freeentry.length = len
			freeTracker.size += len
			memEntries.freeSize += len
		}
	}
}

func linkCacheFreeToAlloc(memEntries *MemEntrieByType) {

	var freeTracker *MemEntryTracker

	freeTracker = &memEntries.kmem_cache_free
	for _, freeentry := range freeTracker.entries {
		if freeentry.ptr == 0 {
			//kfree with zero is valid in kernel
			continue
		}
		len := LinkFreeEntryToAlloc(memEntries,	&memEntries.kmem_cache_alloc, freeentry)
		if len != 0 {
			freeentry.length = len
			freeTracker.size += len
			memEntries.freeSize += len
		}
	}
}

func BuildMemEntries(trace_file string, pid int32) (*MemEntrieByType, error) {

	var tracker *MemEntryTracker

	memEntries := new(MemEntrieByType)
	IntMemTracker(&memEntries.kmalloc, "kmalloc")
	IntMemTracker(&memEntries.kmalloc_node, "kmalloc_node")
	IntMemTracker(&memEntries.kfree, "kfree")
	IntMemTracker(&memEntries.kmem_cache_alloc, "kmem_cache_alloc")
	IntMemTracker(&memEntries.kmem_cache_free, "kmem_cache_free")
	IntMemTracker(&memEntries.mm_page_alloc, "mm_page_alloc")
	IntMemTracker(&memEntries.mm_page_free, "mm_page_free")

	file := FileObject{trace_file, nil}

	data, err := file.Read()
	if err != nil {
		fmt.Println("Fail to read file")
		return nil, err
	}
	lines := strings.Split(data, "\n")
	//skip first 9 lines
	lines = lines[9:]

	for i, line := range lines {
		memEntry, err := parseLine(line, i, pid)
		if err != nil {
			continue
		}

		switch memEntry.call_type {
		case "kmalloc":
			tracker = &memEntries.kmalloc
		case "kmalloc_node":
			tracker = &memEntries.kmalloc_node
		case "kfree":
			tracker = &memEntries.kfree
		case "kmem_cache_alloc":
			tracker = &memEntries.kmem_cache_alloc
		case "kmem_cache_free":
			tracker = &memEntries.kmem_cache_free
		case "mm_page_alloc":
			tracker = &memEntries.mm_page_alloc
		case "mm_page_free":
			tracker = &memEntries.mm_page_free
		default:
			tracker = nil
		}
		if tracker == nil {
			continue
		}
		if memEntry.ptr != 0 {
			tracker.count++
			tracker.size += memEntry.length
			tracker.entries = append(tracker.entries, memEntry)
			tracker.ptr_map[memEntry.ptr] =
				append(tracker.ptr_map[memEntry.ptr], memEntry)
			switch memEntry.call_type {
			case "kmalloc":
				memEntries.allocSize += memEntry.length
			case "kmalloc_node":
				memEntries.allocSize += memEntry.length
			case "kmem_cache_alloc":
				memEntries.allocSize += memEntry.length
			}
		} else {
			if memEntry.call_type == "mm_page_alloc" {
				tracker.count++
				tracker.size += memEntry.length
				memEntries.pageAllocSize += memEntry.length
			} else if memEntry.call_type == "mm_page_free" {
				tracker.count++
				tracker.size += memEntry.length
				memEntries.pageFreeSize += memEntry.length
			}
		}
	}

	linkFreeToAlloc(memEntries)
	linkCacheFreeToAlloc(memEntries)

	return memEntries, nil
}
