#!/bin/bash

cat /sys/kernel/debug/tracing/trace > trace.txt
echo 0 > /sys/kernel/debug/tracing/tracing_on
echo  > /sys/kernel/debug/tracing/trace
