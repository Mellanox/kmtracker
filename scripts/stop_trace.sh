#!/bin/bash

echo 0 > /sys/kernel/debug/tracing/tracing_on
cat /sys/kernel/debug/tracing/trace > trace.txt
echo  > /sys/kernel/debug/tracing/trace
