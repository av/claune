#!/bin/bash
PATH="/home/everlier/go/bin:$PATH" go build -o claune_test_bin ./cmd/claune

# Force aplay usage for testing
rm -rf fake_bin
mkdir -p fake_bin
ln -s /bin/false fake_bin/paplay
ln -s /bin/false fake_bin/pw-play

export PATH="$PWD/fake_bin:$PATH"

for i in {1..5}; do
  ./claune_test_bin play cli:done > out_$i.log 2>&1 &
  pids[${i}]=$!
done

for pid in ${pids[*]}; do
  wait $pid
done

echo "Concurrent aplay test complete"
