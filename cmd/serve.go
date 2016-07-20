package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"net/url"
	"net/http"
	"strings"
	"net/http/httputil"
	"math/rand"
	"strconv"
)

var Port int

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run reverse proxy server",
	Run: func(cmd *cobra.Command, args []string) {
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
		http.ListenAndServe(":" + strconv.Itoa(Port), proxy)
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")
	serveCmd.PersistentFlags().IntVarP(&Port, "port", "p", 9090, "port on which the server will listen")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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
