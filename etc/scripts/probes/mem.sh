#!/bin/bash

# is MemAvailable supported?
ma_supported=$(grep "MemAvailable:" /proc/meminfo)

function meminfo_fmt() {
    val=$(grep "^$1:" /proc/meminfo)
    val=$(echo "$val" | awk '{printf("%i\n", $2/1024)}')
    echo $val
}

if [ -z "$ma_supported" ]; then
    mem=$(free -m | grep '^Mem')
    swap=$(free -m | grep '^Swap')

    mem_total_mb=$(echo $mem | cut -d\  -f2)
    mem_free_mb=$(echo $mem | cut -d\  -f4)
    mem_cached_mb=$(echo $mem | cut -d\  -f7)
    mem_buffers_mb=$(echo $mem | cut -d\  -f6)

    mem_used_ratio=$(echo "$mem" | awk '{printf("%.2f\n", $3/$2);}')
    mem_buffcache_mb=$(($mem_cached_mb + $mem_buffers_mb))
    mem_allocated_mb=$(($mem_total_mb - $mem_free_mb - $mem_buffcache_mb))
    mem_available=$(($mem_total_mb - $mem_allocated_mb))

    swap_total_mb=$(echo $swap | cut -d\  -f2)
    swap_free_mb=$(echo $swap | cut -d\  -f4)
    swap_used_mb=$(echo $swap | cut -d\  -f3)
    swap_used_ratio=$(echo "$swap" | awk '{printf("%.2f\n", $3/$2);}')
else
    mem_total_mb=$(meminfo_fmt MemTotal)
    mem_free_mb=$(meminfo_fmt MemFree)
    mem_used_mb=$(( $mem_total_mb - $mem_free_mb ))
    mem_available=$(meminfo_fmt MemAvailable)
    mem_used_ratio=$(echo $mem_used_mb $mem_total_mb | awk '{printf("%.2f", $1/$2);}')
    mem_buffers_mb=$(meminfo_fmt Buffers)
    mem_cached_mb=$(meminfo_fmt Cached)

    swap_total_mb=$(meminfo_fmt SwapTotal)
    swap_free_mb=$(meminfo_fmt SwapFree)
    swap_used_mb=$(( $swap_total_mb - $swap_free_mb ))
    swap_used_ratio=$(echo "$swap_used_mb" "$swap_free_mb" | awk '{printf("%.2f\n", $1/$2);}')
fi

mem_buffcache_mb=$(($mem_cached_mb + $mem_buffers_mb))
mem_buffcache_ratio=$(echo $mem_total_mb $mem_buffcache_mb\
    | awk '{printf("%.2f\n", $2/$1);}')


echo "MEM_TOTAL_MB:" $mem_total_mb
#sleep 10
echo "MEM_FREE_MB:" $mem_free_mb
echo "MEM_AVAILABLE_MB:" $mem_available
echo "MEM_USED_RATIO:" $mem_used_ratio
echo "MEM_BUFFCACHE_MB:" $mem_buffcache_mb
echo "MEM_BUFFCACHE_RATIO:" $mem_buffcache_ratio
echo "SWAP_TOTAL_MB:" $swap_total_mb
echo "SWAP_FREE_MB:" $swap_free_mb
echo "SWAP_USED_MB:" $swap_used_mb
echo "SWAP_USED_RATIO:" $swap_used_ratio
