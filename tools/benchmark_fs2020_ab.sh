#!/bin/bash
ps -ef | grep -i __debug_bin |  grep -v grep  | awk '{print $2}' | xargs pmap -x | tail -n 1 | awk '{print $4}'
go tool pprof -png http://localhost:8080/debug/pprof/heap > scratch/mrc_fs2020_0.png
ab -c 2 -n 5 -s 300 localhost:8080/test/fs2020
ps -ef | grep -i __debug_bin |  grep -v grep  | awk '{print $2}' | xargs pmap -x | tail -n 1 | awk '{print $4}'
go tool pprof -png http://localhost:8080/debug/pprof/heap > scratch/mrc_fs2020_1.png