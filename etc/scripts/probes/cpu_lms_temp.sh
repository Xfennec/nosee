#!/bin/bash

# this script use lm_sensors to average temperature of all CPU cores
# Required sensors output format:
# ...
# Core 0:         +33.0°C  (high = +82.0°C, crit = +102.0°C)
# Core 1:         +32.0°C  (high = +82.0°C, crit = +102.0°C)

sensors | awk '
BEGIN {
    total = 0
    cores = 0
    high = 999
    crit = 999
}
/^Core/ {
    if (match($0, /\+([0-9.]+)°C.*\+([0-9.]+)°C,.*\+([0-9.]+)°C/, g) > 0) {
	total += g[1]
	high = (g[2] < high ? g[2] : high)
	crit = (g[3] < crit ? g[3] : crit)
	cores++
    } else if (match($0, /\+([0-9.]+)°C/, g) > 0) {
	total += g[1]
	cores++
    }
}
END {
    printf("TEMP: %f\n", total / cores)
    printf("HIGH: %f\n", high)
    printf("CRIT: %f\n", crit)
}
'
