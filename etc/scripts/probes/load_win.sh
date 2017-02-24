#!/bin/bash

if [ $(uname -o) != "Cygwin" ]; then
    (>&2 echo "Cygwin needed")
    exit 1
fi

pql=$(wmic path Win32_PerfFormattedData_PerfOS_System get ProcessorQueueLength | awk 'NR==2')
echo "CPU_QUEUE_LEN:" $pql

# select PercentProcessorTime from Win32_PerfFormattedData_PerfOS_Processor where Name = '_Total'

#p=$(wmic path Win32_PerfFormattedData_PerfOS_System get PercentProcessorQueueLength | awk 'NR==2')
#echo "CPU_QUEUE_LEN:" $pql

ppt=$(wmic path Win32_PerfFormattedData_PerfOS_Processor where "Name = '_Total'" get PercentProcessorTime | awk 'NR==2')
echo CPU_PERCENT: $ppt

lp=$(wmic cpu get loadpercentage | awk 'NR==2')
echo CPU_LOAD_PERCENT: $lp

pdt=$(wmic path Win32_PerfFormattedData_PerfDisk_PhysicalDisk where "Name='_Total'" get PercentDiskTime | awk 'NR==2')
echo DISK_PERCENT: $pdt
