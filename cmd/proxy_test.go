package cmd

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"
)

func TestProxyRun(t *testing.T) {
	// Set proxy parameters
	Upstreams = []string{"localhost:9091", "localhost:9092"}
	TTL = 5

	// Init upstreams cache
	db, err := bolt.Open("/tmp/fedpa.db", 0600, nil)
	assert.Nil(t, err)
	defer func() {
		db.Close()
		os.Remove("/tmp/fedpa.db")
	}()

	// Start backends
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		log.Println("--->", req.RemoteAddr, req.URL.String())
	})
	go http.ListenAndServe(":9091", nil)
	go http.ListenAndServe(":9092", nil)

	// Send HTTP request to proxy
	req, _ := http.NewRequest("GET", "", nil)
	w := httptest.NewRecorder()
	proxy := NewMultipleHostReverseProxy(db)
	proxy.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}
