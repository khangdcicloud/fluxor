#!/bin/bash
# Run the simple loadtest server

echo "Starting simple FastHTTP server..."
echo "Server will run on http://localhost:8080"
echo "Press Ctrl+C to stop the server"
echo ""

# Check if executable exists, if not build it
if [ ! -f "simple_server" ]; then
    echo "Building server..."
    go build -o simple_server simple_server.go
    if [ $? -ne 0 ]; then
        echo "Build failed!"
        exit 1
    fi
fi

# Run the server
./simple_server

