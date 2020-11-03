#!/bin/bash
ps -ef | grep -i __debug_bin |  grep -v grep  | awk '{print $2}' | xargs pmap -x | tail -n 1 | awk '{print $4}'
go tool pprof -png http://localhost:8080/debug/pprof/heap > scratch/mrc_curl_0.png
for i in {1..5}
do
    curl -s -o /dev/null localhost:8080/test/sws
    ps -ef | grep -i __debug_bin |  grep -v grep  | awk '{print $2}' | xargs pmap -x | tail -n 1 | awk '{print $4}'
    go tool pprof -png http://localhost:8080/debug/pprof/heap > scratch/mrc_curl_$i.png
done
#ab -c 2 -n 5 -s 300 localhost:8080/test/fs2020
