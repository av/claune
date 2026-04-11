#!/bin/bash

# Launch the HTTP server in the background
echo "Starting HTTP server on port 8080..."
cd website && python3 -m http.server 8080 > /dev/null 2>&1 &
SERVER_PID=$!

# Wait for the server to start
sleep 2

# Check index.html
echo "Testing index.html..."
if curl -s -f http://localhost:8080/index.html > /dev/null; then
    echo "index.html loaded successfully."
else
    echo "Error: index.html failed to load."
    kill $SERVER_PID
    exit 1
fi

# Check splash.html
echo "Testing splash.html..."
if curl -s -f http://localhost:8080/splash.html > /dev/null; then
    echo "splash.html loaded successfully."
else
    echo "Error: splash.html failed to load."
    kill $SERVER_PID
    exit 1
fi

# Check nav.html
echo "Testing nav.html..."
if curl -s -f http://localhost:8080/nav.html > /dev/null; then
    echo "nav.html loaded successfully."
else
    echo "Error: nav.html failed to load."
    kill $SERVER_PID
    exit 1
fi

echo "All tests passed!"
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null
exit 0
