# Go Proxy Server

This is a simple HTTP/HTTPS proxy server implemented in Go. It supports caching, rate limiting, and logging of all events.

## Features

- **HTTP/HTTPS Proxy**: Handles both HTTP and HTTPS requests.
- **Caching**: Caches responses to reduce load on upstream servers.
- **Rate Limiting**: Limits the number of requests per client to 60 requests per minute.
- **Logging**: Logs all events, including cache hits, request handling, and rate limiting to a specified log file.

## Getting Started

### Prerequisites

- Go 1.15 or later

### Installation

1. Clone the repository:
   ```sh
   git clone https://github.com/Tim-McCann/proxy-server.git
   cd proxy-server
   ```
2. Build the project:
    ```sh
    go build -o proxy_server
    ```
3. Run the server
    ```sh
    cd server
    go run server.go
    ```
4. To get a custom named logfile, run:
    ```sh
    go run server.go -logfile=custom.log
    ```
5. To test the server, run the test file while the server is running:
    ```sh
    cd test
    go run test-proxy.go
    ```


## License 

MIT
