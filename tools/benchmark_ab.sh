#!/bin/bash
GAME="$1"
if [ -z "$GAME" ]
then
    echo "Usage: $0 [game]"
    exit 1
fi

# Run MRC
CMD="go run . -d -d testdata"
echo "Running: $CMD"
$CMD &
if [ $? -ne 0 ]
then
    echo "ERROR: Failed to run '$CMD'"
    exit 1
fi
# Let it startup
sleep 5

# Get MRC pid
PID=`ps f | grep metarefcard | grep -v grep | awk {'print $1'}`
echo "PID $PID"

# Report memory
MEM=`pmap -x $PID | tail -n 1 | awk '{print $4}'`
echo "RSS Mem ${MEM}kb"

# Get first heap
go tool pprof -png http://localhost:8080/debug/pprof/heap > scratch/mrc_ab_0.png
ab -c 4 -n 10 -s 300 localhost:8080/test/$GAME
# Report memory
MEM=`pmap -x $PID | tail -n 1 | awk '{print $4}'`
echo "RSS Mem ${MEM}kb"
# Get next heap
go tool pprof -png http://localhost:8080/debug/pprof/heap > scratch/mrc_ab_1.png

# Stop MRC
kill $PID
