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
	"time"

	"github.com/boltdb/bolt"
	"github.com/spf13/cobra"
)

// Upstream represents upstream target with timestamp
type Upstream struct {
	Target    url.URL   `json:"target"`
	Timestamp time.Time `json:"time"`
}

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
			_, err := tx.CreateBucket([]byte("Upstreams"))
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
	proxyCmd.PersistentFlags().StringSliceVarP(&upstreams, "upstreams", "u", nil,
		"Upstream list in form of 'host1:port1,host2:port2'")
}

// NewMultipleHostReverseProxy creates a reverse proxy that will randomly
// select a host from the passed `targets`
func NewMultipleHostReverseProxy(db *bolt.DB, targets []*url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		ip := strings.Split(req.RemoteAddr, ":")[0]
		var byt []byte
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("Upstreams"))
			byt = b.Get([]byte(ip))
			return nil
		})

		var upstream *Upstream
		if byt != nil {
			if err := json.Unmarshal(byt, &upstream); err != nil {
				log.Printf("Error: %v", err)
			}
			log.Printf("Upstream [%v] with timestamp [%v] for [%s] is found in cache\n",
				upstream.Target.Host, upstream.Timestamp, ip)
		} else {
			upstream = &Upstream{
				Target:    LoadBalance(targets),
				Timestamp: time.Now(),
			}
			db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("Upstreams"))
				encoded, err := json.Marshal(upstream)
				if err != nil {
					return err
				}
				return b.Put([]byte(ip), encoded)
			})
			log.Printf("Upstream [%v] with timestamp [%v] for [%s] is cached",
				upstream.Target.Host, upstream.Timestamp, ip)
		}

		req.URL.Scheme = upstream.Target.Scheme
		req.URL.Host = upstream.Target.Host
		req.URL.Path = singleJoiningSlash(upstream.Target.Path, req.URL.Path)
	}
	return &httputil.ReverseProxy{Director: director}
}

// LoadBalance defines balancing logic
func LoadBalance(targets []*url.URL) url.URL {
	return *targets[rand.Int()%len(targets)]
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
