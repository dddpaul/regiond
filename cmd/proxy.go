package cmd

import (
	"encoding/json"
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

	"smilenet.ru/fedpa/cache"
)

// Upstream represents upstream target with timestamp
type Upstream struct {
	Target    url.URL   `json:"target"`
	Timestamp time.Time `json:"time"`
}

// Upstreams holds list of strings in form of 'host1:port1'
var Upstreams []string

// TTL holds cache record time-to-live in nanoseconds
var TTL int64

const df = "2006-01-02 15:04:05 MST"

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run reverse proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := bolt.Open("fedpa.db", 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		proxy := NewMultipleHostReverseProxy(db)
		http.ListenAndServe(":"+strconv.Itoa(port), proxy)
	},
}

func init() {
	RootCmd.AddCommand(proxyCmd)
	proxyCmd.PersistentFlags().StringSliceVarP(&Upstreams, "upstreams", "u", nil, "Upstream list in form of 'host1:port1,host2:port2'")
	proxyCmd.PersistentFlags().Int64VarP(&TTL, "ttl", "t", 3600, "Cache record time-to-live in seconds")
}

// NewMultipleHostReverseProxy creates a reverse proxy that will randomly
// select a host from the passed `targets`
func NewMultipleHostReverseProxy(db *bolt.DB) *httputil.ReverseProxy {
	cache.Create(db)
	targets := toUrls(Upstreams)
	director := func(req *http.Request) {
		ip := strings.Split(req.RemoteAddr, ":")[0]
		var upstream *Upstream
		newUpstream := false
		if byt := cache.Get(db, ip); byt != nil {
			if err := json.Unmarshal(byt, &upstream); err != nil {
				log.Printf("Error: %v", err)
			}
			if upstream.Timestamp.Add(time.Duration(TTL) * time.Second).After(time.Now()) {
				log.Printf("Upstream [%v] with timestamp [%s] for [%s] is found in cache\n", upstream.Target.Host, upstream.Timestamp.Format(df), ip)
			} else {
				// Upstream record in cache is too old
				cache.Del(db, ip)
				newUpstream = true
			}
		} else {
			// No upstream record in cache
			newUpstream = true
		}
		if newUpstream {
			upstream = &Upstream{
				Target:    LoadBalance(targets),
				Timestamp: time.Now(),
			}
			encoded, err := json.Marshal(upstream)
			if err != nil {
				log.Printf("Error: %v", err)
			}
			cache.Put(db, ip, encoded)
			log.Printf("Upstream [%v] with timestamp [%s] for [%s] is cached", upstream.Target.Host, upstream.Timestamp.Format(df), ip)
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

// Converts list of upstreams to the list of URLs
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

// Taken from net/http/httputil/reverseproxy.go
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
