#!/bin/bash
PATH="/home/everlier/go/bin:$PATH" go build -o claune_test_bin ./cmd/claune

# Wipe cache to ensure they both try to extract
rm -rf ~/.cache/claune

# Start 10 passthroughs that just exit immediately
for i in {1..10}; do
  ./claune_test_bin --help >/dev/null &
  pids[${i}]=$!
done

for pid in ${pids[*]}; do
  wait $pid
  if [ $? -ne 0 ]; then
    echo "Process $pid failed"
    exit 1
  fi
done

echo "All passed"
