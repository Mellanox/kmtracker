#!/bin/bash
#start tracing for a given pid.

echo "starting trace on pid " $1

echo 0 > /sys/kernel/debug/tracing/tracing_on
echo > /sys/kernel/debug/tracing/trace

echo kmem:* > /sys/kernel/debug/tracing/set_event
echo $1 > /sys/kernel/debug/tracing/set_event_pid
echo 1 >  /sys/kernel/debug/tracing/tracing_on
