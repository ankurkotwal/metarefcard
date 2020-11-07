#!/bin/bash
GAME="$1"
if [ -z "$GAME" ]
then
    echo "Usage: $0 [game]"
    exit 1
fi

# Get game files
GAME_ARGS=""
for f in `find testdata/$GAME -maxdepth 1 -type f`
do
    GAME_ARGS="$GAME_ARGS -$GAME $f"
done

# Run MRC
CMD="go run . -d $GAME_ARGS"
echo "Running: $CMD"
$CMD &
if [ $? -ne 0 ]
then
    echo "ERROR: Failed to run '$CMD'"
fi
# Let it startup
sleep 5

# Get MRC pid
PID=`ps f | grep MetaRefCard | grep -v grep | awk {'print $1'}`
echo "PID $PID"

# Report memory
MEM=`pmap -x $PID | tail -n 1 | awk '{print $4}'`
echo "RSS Mem ${MEM}kb"

# Get first heap
go tool pprof -png http://localhost:8080/debug/pprof/heap > scratch/mrc_curl_0.png
for i in {1..10}
do
    # Hit the endpoint
    curl -s -o /dev/null localhost:8080/test/$GAME
    # Report memory
    MEM=`pmap -x $PID | tail -n 1 | awk '{print $4}'`
    echo "RSS Mem ${MEM}kb"
    # Get next heap
    go tool pprof -png http://localhost:8080/debug/pprof/heap > scratch/mrc_curl_$i.png
done

# Stop MRC
kill $PID