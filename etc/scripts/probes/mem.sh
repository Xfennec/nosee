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
    mem_buffcache_mb=$(($mem_cached_mb + $mem_buffers_mb))

    mem_hardused_mb=$(echo "$mem" | awk '{printf("%.2f\n", $3-$5-$6-$7);}')
    mem_hardused_ratio=$(echo $mem_hardused_mb $mem_total_mb | awk '{printf("%.2f", $1/$2);}')

    mem_available_mb=$(($mem_free_mb + $mem_buffcache_mb))

    swap_total_mb=$(echo $swap | cut -d\  -f2)
    swap_free_mb=$(echo $swap | cut -d\  -f4)
    swap_used_mb=$(echo $swap | cut -d\  -f3)
    if [ $swap_total_mb -eq 0 ]; then
        swap_used_ratio=0
    else
        swap_used_ratio=$(echo "$swap" | awk '{printf("%.2f\n", $3/$2);}')
    fi
else
    mem_total_mb=$(meminfo_fmt MemTotal)
    mem_available_mb=$(meminfo_fmt MemAvailable)
    mem_hardused_mb=$(( $mem_total_mb - $mem_available_mb ))
    mem_hardused_ratio=$(echo $mem_hardused_mb $mem_total_mb | awk '{printf("%.2f", $1/$2);}')
    mem_buffers_mb=$(meminfo_fmt Buffers)
    mem_cached_mb=$(meminfo_fmt Cached)

    swap_total_mb=$(meminfo_fmt SwapTotal)
    swap_free_mb=$(meminfo_fmt SwapFree)
    swap_used_mb=$(( $swap_total_mb - $swap_free_mb ))
    if [ $swap_total_mb -eq 0 ]; then
        swap_used_ratio=0
    else
        swap_used_ratio=$(echo "$swap_used_mb" "$swap_total_mb" | awk '{printf("%.2f\n", $1/$2);}')
    fi
fi

mem_buffcache_mb=$(($mem_cached_mb + $mem_buffers_mb))
mem_buffcache_ratio=$(echo $mem_total_mb $mem_buffcache_mb\
    | awk '{printf("%.2f\n", $2/$1);}')
mem_available_ratio=$(echo $mem_total_mb $mem_available_mb\
    | awk '{printf("%.2f\n", $2/$1);}')

echo "MEM_TOTAL_MB:" $mem_total_mb
echo "MEM_AVAILABLE_MB:" $mem_available_mb
echo "MEM_AVAILABLE_RATIO:" $mem_available_ratio
echo "MEM_HARDUSED_MB:" $mem_hardused_mb
echo "MEM_HARDUSED_RATIO:" $mem_hardused_ratio
echo "MEM_BUFFCACHE_MB:" $mem_buffcache_mb
echo "MEM_BUFFCACHE_RATIO:" $mem_buffcache_ratio
echo "SWAP_TOTAL_MB:" $swap_total_mb
echo "SWAP_FREE_MB:" $swap_free_mb
echo "SWAP_USED_MB:" $swap_used_mb
echo "SWAP_USED_RATIO:" $swap_used_ratio
