package cmd

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"
)

func TestProxyRun(t *testing.T) {
	// Start backends
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		log.Println("--->", req.RemoteAddr, req.URL.String())
	})
	go http.ListenAndServe(":9091", nil)
	go http.ListenAndServe(":9092", nil)

	// Set proxy parameters
	Upstreams = []string{"localhost:9091", "localhost:9092"}
	TTL = 5

	// Setup upstreams cache
	db, err := bolt.Open("/tmp/fedpa.db", 0600, nil)
	assert.Nil(t, err)
	defer func() {
		db.Close()
		os.Remove("/tmp/fedpa.db")
	}()

	// Setup proxy
	proxy := NewXffProxy(NewMultipleHostProxy(db))

	for i := 1; i <= 3; i++ {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "", nil)
		assert.Nil(t, err)
		req.Header.Set("X-Forwarded-For", "20.0.0."+strconv.Itoa(i))
		proxy.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}
}
