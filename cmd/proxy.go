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

// Env holds datasources and other environment
type Env struct {
	Blt *bolt.DB
	Ora *sql.DB
}

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

var env *Env

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
		env = &Env{
			Blt: blt,
			Ora: ora,
		}
		proxy := NewXffProxy(NewMultipleHostProxy(env))
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
func NewMultipleHostProxy(env *Env) *httputil.ReverseProxy {
	cache.Create(env.Blt)
	targets := toUrls(Upstreams)

	director := func(req *http.Request) {
		ip := strings.Split(req.RemoteAddr, ":")[0]

		// Prepare statement here to be able to close it by defer in case of database unavailability
		var stmt *sql.Stmt
		var err error
		if env.Ora != nil {
			stmt, err = env.Ora.Prepare("SELECT region FROM ip_to_region WHERE rownum = 1 AND ip = :1")
			if err != nil {
				log.Printf("[%s] - Error: %v\n", ip, err)
			}
			defer func() {
				// Recover from panic caused by statement close when database is unavailable
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[%s] - Recovered from: %v\n", ip, r)
					}
				}()
				stmt.Close()
			}()
		}

		var upstream *Upstream
		newUpstream := false
		if byt := cache.Get(env.Blt, ip); byt != nil {
			if err := json.Unmarshal(byt, &upstream); err != nil {
				log.Printf("[%s] - Error: %v\n", ip, err)
			}
			if upstream.Timestamp.Add(time.Duration(TTL) * time.Second).After(time.Now()) {
				// log.Printf("Upstream [%v] with timestamp [%s] for [%s] is found in cache\n", upstream.Target.Host, upstream.Timestamp.Format(df), ip)
			} else {
				// Upstream record in cache is too old
				cache.Del(env.Blt, ip)
				newUpstream = true
			}
		} else {
			// No upstream record in cache
			newUpstream = true
		}

		if newUpstream {
			upstream = &Upstream{
				Target:    *LoadBalance(targets, ip, stmt),
				Timestamp: time.Now(),
			}
			encoded, err := json.Marshal(upstream)
			if err != nil {
				log.Printf("[%s] - Error: %v\n", ip, err)
			}
			cache.Put(env.Blt, ip, encoded)
			// log.Printf("Upstream [%v] with timestamp [%s] for [%s] is cached", upstream.Target.Host, upstream.Timestamp.Format(df), ip)
		}

		req.URL.Scheme = upstream.Target.Scheme
		req.URL.Host = upstream.Target.Host
		req.URL.Path = singleJoiningSlash(upstream.Target.Path, req.URL.Path)
	}

	log.Printf("Reverse proxy is listening on port %d for upstreams %v with TTL %d seconds", port, targets, TTL)
	return &httputil.ReverseProxy{Director: director}
}

// LoadBalance defines balancing logic.
// Returns upstream based on value from Oracle table.
func LoadBalance(targets []*url.URL, ip string, stmt *sql.Stmt) *url.URL {
	// Returns random upstrem if Oracle database is not used
	if stmt == nil {
		return targets[rand.Int()%len(targets)]
	}

	var region int
	err := stmt.QueryRow(ip).Scan(&region)
	if err != nil {
		log.Printf("[%s] - Error: %v\n", ip, err)
		// Use first upstream on error
		return targets[0]
	}

	return targets[region-1]
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
