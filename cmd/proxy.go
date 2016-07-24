package cmd

import (
	"database/sql"
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
	_ "github.com/mattn/go-oci8" // Oracle driver
	"github.com/sebest/xff"
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

// BoltFn is Bolt filename (local caching key-value storage)
var BoltFn string

// OraConnStr is Oracle connection string in form of 'user/pass@host/sid'
var OraConnStr string

const df = "2006-01-02 15:04:05 MST"

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run reverse proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		blt, err := bolt.Open(BoltFn, 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		ora, err := sql.Open("oci8", OraConnStr)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			blt.Close()
			ora.Close()
		}()
		proxy := NewXffProxy(NewMultipleHostProxy(blt, ora))
		http.ListenAndServe(":"+strconv.Itoa(port), proxy)
	},
}

func init() {
	RootCmd.AddCommand(proxyCmd)
	proxyCmd.PersistentFlags().StringSliceVarP(&Upstreams, "upstreams", "u", nil, "Upstream list in form of 'host1:port1,host2:port2'")
	proxyCmd.PersistentFlags().Int64VarP(&TTL, "ttl", "t", 3600, "Cache record time-to-live in seconds")
	proxyCmd.PersistentFlags().StringVarP(&OraConnStr, "oracle", "o", "system/oracle@localhost/xe", "Oracle connection string in form of 'user/pass@host/sid'")
	proxyCmd.PersistentFlags().StringVarP(&BoltFn, "bolt", "b", "fedpa.db", "Bolt caching key-value storage filename")
}

// NewXffProxy wraps reverse proxy with X-Forwarded-For handler
func NewXffProxy(p *httputil.ReverseProxy) http.Handler {
	xffmw, err := xff.Default()
	if err != nil {
		log.Fatal(err)
	}
	return xffmw.Handler(p)
}

// NewMultipleHostProxy creates a reverse proxy that will randomly
// select a host from the passed `targets`
func NewMultipleHostProxy(blt *bolt.DB, ora *sql.DB) *httputil.ReverseProxy {
	cache.Create(blt)
	targets := toUrls(Upstreams)
	director := func(req *http.Request) {
		ip := strings.Split(req.RemoteAddr, ":")[0]
		var upstream *Upstream
		newUpstream := false
		if byt := cache.Get(blt, ip); byt != nil {
			if err := json.Unmarshal(byt, &upstream); err != nil {
				log.Printf("Error: %v\n", err)
			}
			if upstream.Timestamp.Add(time.Duration(TTL) * time.Second).After(time.Now()) {
				log.Printf("Upstream [%v] with timestamp [%s] for [%s] is found in cache\n", upstream.Target.Host, upstream.Timestamp.Format(df), ip)
			} else {
				// Upstream record in cache is too old
				cache.Del(blt, ip)
				newUpstream = true
			}
		} else {
			// No upstream record in cache
			newUpstream = true
		}
		if newUpstream {
			target := LoadBalance(targets, ip, ora)
			upstream = &Upstream{
				Target:    *target,
				Timestamp: time.Now(),
			}
			encoded, err := json.Marshal(upstream)
			if err != nil {
				log.Printf("Error: %v\n", err)
			}
			cache.Put(blt, ip, encoded)
			log.Printf("Upstream [%v] with timestamp [%s] for [%s] is cached", upstream.Target.Host, upstream.Timestamp.Format(df), ip)
		}

		req.URL.Scheme = upstream.Target.Scheme
		req.URL.Host = upstream.Target.Host
		req.URL.Path = singleJoiningSlash(upstream.Target.Path, req.URL.Path)
	}
	log.Printf("Reverse proxy is listening on port %d for upstreams %v with TTL %d seconds", port, targets, TTL)
	return &httputil.ReverseProxy{Director: director}
}

// LoadBalance defines balancing logic.
// Returns random target if Oracle database is not used.
// Use target based on value from Oracle table otherwise.
func LoadBalance(targets []*url.URL, ip string, ora *sql.DB) *url.URL {
	if ora == nil {
		return targets[rand.Int()%len(targets)]
	}

	// Recover from Oracle driver panic when database is unavailable
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from %v\n", r)
		}
	}()

	region := selectRegion(ip, ora)
	return targets[region]
}

// Returns offset for upstream list based on region code
func selectRegion(ip string, ora *sql.DB) int {
	rows, err := ora.Query("SELECT region FROM ip_to_region WHERE rownum = 1 AND ip = :1", ip)
	defer rows.Close()
	if err != nil {
		log.Printf("Error: %v\n", err)
		return 0 // Use first upstream on error
	}
	var region int
	for rows.Next() {
		rows.Scan(&region)
	}
	return region - 1
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
