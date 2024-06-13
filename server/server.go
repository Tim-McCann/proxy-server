package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	cache        = make(map[string][]byte)
	cacheMutex   = sync.Mutex{}
	clients      = make(map[string]int)
	clientsMux   = sync.Mutex{}
	logFile      *os.File
	logFileMutex = sync.Mutex{}
)

const (
	maxRequestsPerMinute = 60
)

func cacheKey(u *url.URL) string {
	h := sha1.New()
	h.Write([]byte(u.String()))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func logEvent(format string, v ...interface{}) {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()
	log.Printf(format, v...)
	if logFile != nil {
		logFile.WriteString(fmt.Sprintf(format+"\n", v...))
	}
}

func handleRequestAndCache(res http.ResponseWriter, req *http.Request) {
	start := time.Now()

	if !strings.HasPrefix(req.RequestURI, "http://") && !strings.HasPrefix(req.RequestURI, "https://") {
		req.RequestURI = "http://" + req.Host + req.RequestURI
	}

	parsedURL, err := url.Parse(req.RequestURI)
	if err != nil {
		http.Error(res, "Bad request", http.StatusBadRequest)
		return
	}

	key := cacheKey(parsedURL)

	cacheMutex.Lock()
	if cachedResp, found := cache[key]; found {
		cacheMutex.Unlock()
		logEvent("CACHE HIT: %s", req.RequestURI)
		res.Write(cachedResp)
		logEvent("Served %s in %v\n", req.RequestURI, time.Since(start))
		return
	}
	cacheMutex.Unlock()

	proxyReq, err := http.NewRequest(req.Method, parsedURL.String(), req.Body)
	if err != nil {
		http.Error(res, "Failed to create request", http.StatusInternalServerError)
		return
	}

	for header, values := range req.Header {
		for _, value := range values {
			proxyReq.Header.Add(header, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(res, "Failed to forward request", http.StatusInternalServerError)
		logEvent("Failed to forward request: %s, error: %v", req.RequestURI, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(res, "Failed to read response body", http.StatusInternalServerError)
		logEvent("Failed to read response body: %s, error: %v", req.RequestURI, err)
		return
	}

	cacheMutex.Lock()
	cache[key] = body
	cacheMutex.Unlock()

	for header, values := range resp.Header {
		for _, value := range values {
			res.Header().Add(header, value)
		}
	}

	res.WriteHeader(resp.StatusCode)
	res.Write(body)
	logEvent("Served %s in %v\n", req.RequestURI, time.Since(start))
}

func handleConnect(res http.ResponseWriter, req *http.Request) {
	destConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		http.Error(res, "Failed to connect to destination", http.StatusServiceUnavailable)
		logEvent("Failed to connect to destination: %s, error: %v", req.Host, err)
		return
	}
	defer destConn.Close()

	res.WriteHeader(http.StatusOK)
	hijacker, ok := res.(http.Hijacker)
	if !ok {
		http.Error(res, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(res, "Failed to hijack connection", http.StatusServiceUnavailable)
		logEvent("Failed to hijack connection: %s, error: %v", req.Host, err)
		return
	}
	defer clientConn.Close()

	go io.Copy(destConn, clientConn)
	io.Copy(clientConn, destConn)
}

func extractIP(remoteAddr string) string {
	if strings.Contains(remoteAddr, ":") {
		host, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			return remoteAddr
		}
		return host
	}
	return remoteAddr
}

func rateLimiter(next http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		clientIP := extractIP(req.RemoteAddr)
		clientsMux.Lock()
		count := clients[clientIP]
		logEvent("Client %s has made %d requests", clientIP, count)
		if count >= maxRequestsPerMinute {
			clientsMux.Unlock()
			http.Error(res, "Too Many Requests", http.StatusTooManyRequests)
			logEvent("Rate limit exceeded for client %s", clientIP)
			return
		}
		clients[clientIP] = count + 1
		clientsMux.Unlock()
		next(res, req)
	}
}

func resetRateLimiter() {
	for {
		time.Sleep(1 * time.Minute)
		clientsMux.Lock()
		clients = make(map[string]int)
		clientsMux.Unlock()
	}
}

func main() {
	var logFileName string
	flag.StringVar(&logFileName, "logfile", "proxy.log", "File to log all events")
	flag.Parse()

	var err error
	logFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer logFile.Close()

	go resetRateLimiter()

	http.HandleFunc("/", rateLimiter(handleRequestAndCache))
	http.HandleFunc("/CONNECT", rateLimiter(handleConnect))
	fmt.Println("Proxy server is running on port 8080")
	logEvent("Proxy server started on port 8080")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		logEvent("Error starting server: %v", err)
	}
}
