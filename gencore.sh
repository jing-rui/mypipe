#!/bin/bash

echo '/tmp/core_%e_%p_%t' > /proc/sys/kernel/core_pattern
ulimit -c unlimited
export GOTRACEBACK=crash

cmd=./mypipe
beg=$(date '+%Y-%m-%d-%H-%M-%S.%3N')
mkdir run.$beg; cd run.$beg; cp ../mypipe .

for ((i=0;i<1000000000;i++)); do
    out=$($cmd)
    ret=$?
    now=$(date '+%Y-%m-%d-%H-%M-%S.%3N')
    echo beg=$beg now=$now loop=$i ret=$ret
    if [ "$ret" != 0 ]; then
        exit
    fi
done
