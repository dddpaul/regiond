package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"time"

	"database/sql"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/dddpaul/regiond/cache"
	"github.com/dddpaul/regiond/cmd"
	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
	"net"
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
	blt, err := bolt.Open("/tmp/regiond.db", 0600, nil)
	assert.Nil(t, err)
	defer func() {
		blt.Close()
		os.Remove("/tmp/regiond.db")
	}()

	// Setup proxy
	env := &cmd.Env{
		Blt: blt,
		Ora: nil,
	}
	proxy := cmd.NewXffProxy(cmd.NewMultipleHostProxy(env))

	// Send bunch of HTTP requests, cache must be filled
	for i := 1; i <= REQUESTS; i++ {
		req := prepareRequest(t, "/", i)
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}
	cache1 := cache.PrefixScan(blt, "20.0.0")
	assert.Equal(t, REQUESTS, len(cache1))

	// Send bunch of same HTTP requests, cache must stay the same
	for i := 1; i <= REQUESTS; i++ {
		req := prepareRequest(t, "/", i)
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
		req := prepareRequest(t, "/", i)
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}
	cache3 := cache.PrefixScan(blt, "20.0.0")
	assert.NotEqual(t, cache1, cache3)

	// Send bunch of same HTTP requests, cache must stay the same
	w := httptest.NewRecorder()
	for i := 1; i <= REQUESTS; i++ {
		req := prepareRequest(t, "/", i)
		proxy.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}
	cache4 := cache.PrefixScan(blt, "20.0.0")
	assert.Equal(t, cache3, cache4)
}

func TestProxyIsRequestingOracle(t *testing.T) {
	// Start backends which response with listening port
	h := func(port int) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte(strconv.Itoa(port)))
		}
	}
	mux1 := http.NewServeMux()
	mux1.Handle("/", h(9093))
	mux2 := http.NewServeMux()
	mux2.Handle("/", h(9094))
	go http.ListenAndServe(":9093", mux1)
	go http.ListenAndServe(":9094", mux2)
	time.Sleep(100 * time.Millisecond)

	// Start Oracle server in Docker container
	client, err := docker.NewClientFromEnv()
	assert.Nil(t, err)
	c := createOracleContainer(t, client)
	err = client.StartContainer(c.ID, nil)
	assert.Nil(t, err)
	defer removeOracleContainer(t, client, c)
	time.Sleep(60 * time.Second)

	// Set proxy parameters
	cmd.Upstreams = []string{"localhost:9093", "localhost:9094"}
	cmd.TTL = 1

	// Setup Oracle database client
	ora, err := sql.Open("oci8", "system/oracle@localhost/xe")
	assert.Nil(t, err)

	// Setup proxy
	env := &cmd.Env{
		Blt: nil,
		Ora: ora,
	}
	proxy := cmd.NewXffProxy(cmd.NewMultipleHostProxy(env))

	// Send bunch of HTTP requests, they must be successfully proxied to second backend according to init.sql
	for i := 1; i <= REQUESTS; i++ {
		req := prepareRequest(t, "/", i)
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "9094", w.Body.String())
	}
}

func prepareRequest(t *testing.T, url string, i int) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	assert.Nil(t, err)
	req.RemoteAddr = "10.0.0." + strconv.Itoa(i) + ":4000" + strconv.Itoa(i)
	req.Header.Set("X-Forwarded-For", "20.0.0."+strconv.Itoa(i))
	return req
}

func createOracleContainer(t *testing.T, client *docker.Client) *docker.Container {
	portBindings := map[docker.Port][]docker.PortBinding{"1521/tcp": {{HostIP: "0.0.0.0", HostPort: "1521"}}}
	client.CreateVolume(docker.CreateVolumeOptions{})
	c, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: "regiond-oracle",
		Config: &docker.Config{
			Image: "wnameless/oracle-xe-11g",
		},
		HostConfig: &docker.HostConfig{
			PortBindings: portBindings,
			Binds: []string{
				os.Getenv("GOPATH") + "/src/github.com/dddpaul/regiond/sql/:/docker-entrypoint-initdb.d/",
			},
		},
	})
	assert.Nil(t, err)
	return c
}

func removeOracleContainer(t *testing.T, client *docker.Client, c *docker.Container) {
	err := client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    c.ID,
		Force: true,
	})
	assert.Nil(t, err)
}

// waitReachable waits for hostport to became reachable for the maxWait time.
func waitReachable(hostport string, maxWait time.Duration) error {
	done := time.Now().Add(maxWait)
	for time.Now().Before(done) {
		c, err := net.Dial("tcp", hostport)
		if err == nil {
			c.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("cannot connect %v for %v", hostport, maxWait)
}

// waitStarted waits for a container to start for the maxWait time.
func waitStarted(client *docker.Client, id string, maxWait time.Duration) error {
	done := time.Now().Add(maxWait)
	for time.Now().Before(done) {
		c, err := client.InspectContainer(id)
		if err != nil {
			break
		}
		if c.State.Running {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("cannot start container %s for %v", id, maxWait)
}
