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

const df = "2006-01-02 15:04:05 MST"

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run reverse proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		blt, err := bolt.Open("fedpa.db", 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		ora, err := sql.Open("oci8", "system/oracle@localhost/xe")
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
				log.Printf("Error: %v", err)
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
			// TODO: Handle error
			target, _ := LoadBalance(targets, ip, ora)
			upstream = &Upstream{
				Target:    *target,
				Timestamp: time.Now(),
			}
			encoded, err := json.Marshal(upstream)
			if err != nil {
				log.Printf("Error: %v", err)
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
func LoadBalance(targets []*url.URL, ip string, ora *sql.DB) (*url.URL, error) {
	if ora == nil {
		return targets[rand.Int()%len(targets)], nil
	}

	rows, err := ora.Query("SELECT region FROM ip_to_region WHERE rownum = 1 AND ip = :1", ip)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	var region int
	for rows.Next() {
		rows.Scan(&region)
	}

	if region == 0 {
		return targets[0], nil
	}

	return targets[region-1], nil
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
