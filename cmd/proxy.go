package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
	"net/url"
	"net/http/httputil"
	"net/http"
	"strconv"
	"math/rand"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run reverse proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Reverse proxy is listening on port %d\n", port)
		proxy := NewMultipleHostReverseProxy([]*url.URL{
			{
				Scheme: "http",
				Host: "localhost:9091",
			},
			{
				Scheme: "http",
				Host: "localhost:9092",
			},
		})
		http.ListenAndServe(":" + strconv.Itoa(port), proxy)
	},
}

func init() {
	RootCmd.AddCommand(proxyCmd)
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

// NewMultipleHostReverseProxy creates a reverse proxy that will randomly
// select a host from the passed `targets`
func NewMultipleHostReverseProxy(targets []*url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		fmt.Println(req.RemoteAddr)
		target := targets[rand.Int() % len(targets)]
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
	}
	return &httputil.ReverseProxy{Director: director}
}
