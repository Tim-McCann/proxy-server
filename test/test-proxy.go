package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	proxyURL          = "http://localhost:8080"
	targetURL         = "http://example.com"
	requestsPerMinute = 2000 // Number of requests to make in one minute
)

func makeRequest(wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{}
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Set the proxy URL
	proxyURL, err := url.Parse(proxyURL)
	if err != nil {
		fmt.Printf("Error parsing proxy URL: %v\n", err)
		return
	}
	client.Transport = &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Request error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %s\n", resp.Status)
}

func main() {
	var wg sync.WaitGroup

	startTime := time.Now()
	for i := 0; i < requestsPerMinute; i++ {
		wg.Add(1)
		go makeRequest(&wg)
		time.Sleep(time.Second * 60 / requestsPerMinute)
	}

	wg.Wait()
	duration := time.Since(startTime)
	fmt.Printf("Completed %d requests in %v\n", requestsPerMinute, duration)
}
