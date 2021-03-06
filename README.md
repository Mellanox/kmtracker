# kmtracker

## overview
Linux Kernel memory tracker

This tool helps users to which kernel functions are allocating and freeing how much memory.
Currently this reporting is for a specified process.

This tool written in golang.

It uses objedump, kallsyms, event tracing and creates database of which precise function allocated/freed memory.
It can track down which kernel API such as kmalloc/kmalloc_node etc used for memory allocation.

Currently it cannot track page allocations such as alloc_page/__free_pages().

## how-to use?
### how to build kmtracker?
```
git clone https://github.com/Mellanox/kmtracker.git
cd kmtracker
make
```

### how to run kmtracker?
```
./scripts/start_trace.sh <pid>
./scripts/stop_trace.sh
./kmtracker <trace_file_name.txt> <pid> <absolute_path_to_vmlinux>
```

Here tracefile is: /sys/kernel/debug/tracing/trace.

pid is: pid whose memory allocations to be tracked.

path to vmlinux: absolute path to vmlinux file. (vmlinuz is not sufficient).

### how to build mmtracker?
```
git clone https://github.com/Mellanox/kmtracker.git
cd kmtracker/mm_tracker
go build .
```

### how to run mm_tracker?
```
./mm_tracker <trace_file_name>
```

For running as verbose mode.
```
./mm_tracker <trace_file_name> -v
```

## TODO
(a) Cross reference free with alloc calls.

## Authors
    Parav Pandit <parav@mellanox.com>
