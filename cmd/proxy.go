package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/spf13/cobra"
)

var upstreams []string

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run reverse proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := bolt.Open("fedpa.db", 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucket([]byte("IpAddresses"))
			if err != nil {
				return fmt.Errorf("Create bucket: %s", err)
			}
			return nil
		})
		targets := toUrls(upstreams)
		log.Printf("Reverse proxy is listening on port %d for upstreams %v\n", port, targets)
		proxy := NewMultipleHostReverseProxy(db, targets)
		http.ListenAndServe(":"+strconv.Itoa(port), proxy)
	},
}

func init() {
	RootCmd.AddCommand(proxyCmd)
	proxyCmd.PersistentFlags().StringSliceVarP(&upstreams, "upstreams", "u", nil, "Upstream list in form of 'host1:port1,host2:port2'")
}

// NewMultipleHostReverseProxy creates a reverse proxy that will randomly
// select a host from the passed `targets`
func NewMultipleHostReverseProxy(db *bolt.DB, targets []*url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		ip := strings.Split(req.RemoteAddr, ":")[0]
		var target *url.URL
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("IpAddresses"))
			json.Unmarshal(b.Get([]byte(ip)), &target)
			return nil
		})

		if target != nil {
			log.Printf("Upstream [%v] for [%s] is found in cache\n", target, ip)
		} else {
			target = LoadBalance(targets)
			db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("IpAddresses"))
				encoded, err := json.Marshal(target)
				if err != nil {
					return err
				}
				return b.Put([]byte(ip), encoded)
			})
			log.Printf("Upstream [%v] for [%s] is cached", target, ip)
		}

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
	}
	return &httputil.ReverseProxy{Director: director}
}

// LoadBalance defines balancing logic
func LoadBalance(targets []*url.URL) *url.URL {
	return targets[rand.Int()%len(targets)]
}

func toUrls(upstreams []string) []*url.URL {
	var urls []*url.URL
	for _, upstream := range upstreams {
		urls = append(urls, &url.URL{
			Scheme: "http",
			Host:   upstream,
		})
	}
	return urls
}

// Copy from net/http/httputil/reverseproxy.go
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
