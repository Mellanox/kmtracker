package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Symbol struct {
	Name         string
	Module       string
	StartAddress uint64 // updated from /proc/kallsyms
	EndAddress   uint64 // updated from <module>.ko
}

type ModuleSymbols struct {
	// key: symbol name
	// value: symbols
	Symbols map[string]*Symbol
}

type KernelSymbols struct {
	ModulesSymbols map[string]*ModuleSymbols
}

// build module list from the kallsyms
func BuildModulesList(kallsyms_map *map[string]*Symbol) map[string]int {

	var count int

	module_list := make(map[string]int)
	for _, sym := range *kallsyms_map {
		if len(sym.Module) != 0 {
			count = module_list[sym.Module]
			count++
			module_list[sym.Module] = count
		}
	}
	return module_list
}

func BuildKallsymsMap() (*KernelSymbols, error) {

	var module_symbols, kernel_symbols int
	var modSymbols *ModuleSymbols

	file := FileObject{"/proc/kallsyms", nil}
	//file := FileObject{ "1.txt", nil }

	data, err := file.Read()
	if err != nil {
		fmt.Println("Fail to read file")
		return nil, err
	}
	array := strings.Split(data, "\n")

	symbols := new(KernelSymbols)
	symbols.ModulesSymbols = make(map[string]*ModuleSymbols)

	for _, line := range array {

		line = strings.Replace(line, "\t", " ", -1)

		words := strings.Split(line, " ")

		if len(words) != 3 && len(words) != 4 {
			continue
		}
		if string(words[1]) != "T" && string(words[1]) != "t" {
			continue
		}

		addr, err := strconv.ParseUint(string(words[0]), 16, 64)
		if err != nil {
			fmt.Println(err)
			continue
		}

		symbol := new(Symbol)
		symbol.StartAddress = addr
		symbol.Name = string(words[2])
		if len(words) == 4 {
			module_name := strings.TrimLeft(string(words[3]), "[")
			module_name = strings.TrimRight(module_name, "]")
			symbol.Module = module_name
			module_symbols++
			//fmt.Println(symbol.Name)
		} else {
			symbol.Module = "linux_kernel"
			kernel_symbols++
		}
		if symbols.ModulesSymbols[symbol.Module] == nil {
			modSymbols = new(ModuleSymbols)
			modSymbols.Symbols = make(map[string]*Symbol)

			symbols.ModulesSymbols[symbol.Module] = modSymbols
		} else {
			modSymbols = symbols.ModulesSymbols[symbol.Module]
		}
		//address->symbol map
		modSymbols.Symbols[string(words[0])] = symbol
	}

	fmt.Println("\tTotal module symbols = ", module_symbols)
	fmt.Println("\tTotal kernel symbols = ", kernel_symbols)
	return symbols, nil
}

func GetModuleKoFileName(module_name string) (string, error) {

	cmdString := "modinfo " + module_name

	output := execShellCmdOutput(cmdString)
	if len(output) == 0 {
		return "", fmt.Errorf("Module not found")
	}
	lines := strings.Split(output, "\n")
	line0 := strings.Trim(lines[0], "filename:")
	line0 = strings.TrimLeft(line0, " ")
	return string(line0), nil
}

// GetModuleSymbolsLen
// Returns map: key(text symbol name) -> value (length of the symbol)
func GetModuleSymbolsLen(module_file string) (map[string]uint64, error) {

	//fmt.Println(module_file)

	objdumpSymbols := make(map[string]uint64)
	cmdString := "objdump -t " + module_file

	output := execShellCmdOutput(cmdString)
	if len(output) == 0 {
		return objdumpSymbols, fmt.Errorf("Module objdump error")
	}
	lines := strings.Split(output, "\n")
	//skip 4 lines
	lines = lines[4:]

	for _, line := range lines {

		line = strings.Replace(line, "\t", " ", -1)
		if strings.Contains(line, ".text..refcount") {
			continue
		}

		if strings.Contains(line, ".init.text") ||
			strings.Contains(line, ".exit.text") ||
			strings.Contains(line, ".text") {
			words := strings.Split(line, " ")

			filtered_entry := make([]string, 6)
			cnt := 0
			for _, word := range words {
				if len(word) == 0 || word == " " {
					continue
				}
				filtered_entry[cnt] = word
				cnt++
			}
			symLen, err := strconv.ParseUint(filtered_entry[4], 16, 64)
			if err != nil {
				continue
			}
			objdumpSymbols[filtered_entry[5]] = symLen
		}
	}

	return objdumpSymbols, nil
}

func update_modules_symbols_len(symbols *KernelSymbols) error {
	var objdumpSymbols map[string]uint64
	var err2 error
	var total_update_cnt int
	var update_cnt int

	fmt.Println("Updating modules symbols length")

	for k, _ := range symbols.ModulesSymbols {
		if k == "linux_kernel" {
			continue
		}
		file, err := GetModuleKoFileName(k)
		if err != nil {
			continue
		}
		objdumpSymbols, err2 = GetModuleSymbolsLen(file)
		if err2 != nil {
			return err2
		}
		update_cnt = 0
		for _, ksym := range symbols.ModulesSymbols[k].Symbols {
			if ksym.EndAddress != 0 {
				continue
			}
			for name, lensym := range objdumpSymbols {
				if ksym.Name == name && k == ksym.Module {
					ksym.EndAddress = ksym.StartAddress + lensym - 1
					//fmt.Println("name = ", ksym.Name)
					update_cnt++
					total_update_cnt++
				}
			}
		}
		//fmt.Printf("module %v = symbols resolved = %d\n",
		//		k, update_cnt)
	}
	fmt.Printf("\tTotal update cnt for modules = %d\n", total_update_cnt)
	fmt.Println("Updating modules symbols length done")
	return nil
}

func update_kernel_symbols_len(kernel_file string,
	kallsyms_map *ModuleSymbols) error {
	var objdumpSymbols map[string]uint64
	var total_kernel_update_cnt int
	var err3 error

	// take kernel file as optional.
	if len(kernel_file) == 0 {
		return nil
	}

	fmt.Println("Updating kernel symbols length")
	objdumpSymbols, err3 = GetModuleSymbolsLen(kernel_file)
	if err3 != nil {
		return err3
	}
	for _, ksym := range kallsyms_map.Symbols {
		if ksym.EndAddress != 0 {
			continue
		}
		if objdumpSymbols[ksym.Name] != 0 {
			ksym.EndAddress = ksym.StartAddress + objdumpSymbols[ksym.Name] - 1
			total_kernel_update_cnt++
		}
	}
	fmt.Printf("\tTotal update cnt for kernel = %d\n", total_kernel_update_cnt)
	fmt.Println("Updating kernel symbols length done")
	return nil
}

func GetLiveKernelSymbolMap(kernel_file string) (*KernelSymbols, error) {

	fmt.Println("Building kallsyms map")
	symbols, err := BuildKallsymsMap()
	if err != nil {
		return nil, err
	}
	fmt.Println("Building kallsyms map done")

	fmt.Println("Building modules list")
	// map: key->value,
	//      key: kernel module(driver) name
	//      value: is count of symbols that belong to that object.
	fmt.Println("Building modules list done")

	update_modules_symbols_len(symbols)
	update_kernel_symbols_len(kernel_file,
		symbols.ModulesSymbols["linux_kernel"])

	return symbols, nil
}
