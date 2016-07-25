package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"time"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"
	"smilenet.ru/fedpa/cache"
	"smilenet.ru/fedpa/cmd"
)

const REQUESTS = 5

func TestProxyIsCachingUpstreams(t *testing.T) {
	// Start backends
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {})
	go http.ListenAndServe(":9091", nil)
	go http.ListenAndServe(":9092", nil)
	time.Sleep(100 * time.Millisecond)

	// Set proxy parameters
	cmd.Upstreams = []string{"localhost:9091", "localhost:9092"}
	cmd.TTL = 1

	// Setup upstreams cache
	blt, err := bolt.Open("/tmp/fedpa.db", 0600, nil)
	assert.Nil(t, err)
	defer func() {
		blt.Close()
		os.Remove("/tmp/fedpa.db")
	}()

	// Setup proxy
	proxy := cmd.NewXffProxy(cmd.NewMultipleHostProxy(blt, nil))

	// Send bunch of HTTP requests, cache must be filled
	for i := 1; i <= REQUESTS; i++ {
		req := prepareRequest(t, i)
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}
	cache1 := cache.PrefixScan(blt, "20.0.0")
	assert.Equal(t, REQUESTS, len(cache1))

	// Send bunch of same HTTP requests, cache must stay the same
	for i := 1; i <= REQUESTS; i++ {
		req := prepareRequest(t, i)
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}
	cache2 := cache.PrefixScan(blt, "20.0.0")
	assert.Equal(t, cache1, cache2)

	// Wait until TTL is expired
	time.Sleep(time.Duration(cmd.TTL) * time.Second)

	// Send bunch of same HTTP requests, cache must be renewed
	for i := 1; i <= REQUESTS; i++ {
		req := prepareRequest(t, i)
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}
	cache3 := cache.PrefixScan(blt, "20.0.0")
	assert.NotEqual(t, cache1, cache3)

	// Send bunch of same HTTP requests, cache must stay the same
	w := httptest.NewRecorder()
	for i := 1; i <= REQUESTS; i++ {
		req := prepareRequest(t, i)
		proxy.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}
	cache4 := cache.PrefixScan(blt, "20.0.0")
	assert.Equal(t, cache3, cache4)
}

func prepareRequest(t *testing.T, i int) *http.Request {
	req, err := http.NewRequest("GET", "", nil)
	assert.Nil(t, err)
	req.RemoteAddr = "10.0.0." + strconv.Itoa(i) + ":4000" + strconv.Itoa(i)
	req.Header.Set("X-Forwarded-For", "20.0.0."+strconv.Itoa(i))
	return req
}