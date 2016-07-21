package cmd

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var upstreams []string

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run reverse proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		targets := toUrls(upstreams)
		fmt.Printf("Reverse proxy is listening on port %d for upstreams %v\n", port, targets)
		proxy := NewMultipleHostReverseProxy(targets)
		http.ListenAndServe(":"+strconv.Itoa(port), proxy)
	},
}

func init() {
	RootCmd.AddCommand(proxyCmd)
	proxyCmd.PersistentFlags().StringSliceVarP(&upstreams, "upstreams", "u", nil, "Upstream list in form of 'host1:port1,host2:port2'")
}

// NewMultipleHostReverseProxy creates a reverse proxy that will randomly
// select a host from the passed `targets`
func NewMultipleHostReverseProxy(targets []*url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		fmt.Println(req.RemoteAddr)
		target := targets[rand.Int()%len(targets)]
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
	}
	return &httputil.ReverseProxy{Director: director}
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
