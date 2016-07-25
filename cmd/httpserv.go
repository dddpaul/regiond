package cmd

import (
	"log"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"
)

var httpservCmd = &cobra.Command{
	Use:   "httpserv",
	Short: "Simple HTTP server for testing",
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("HTTP server is listening on port %d\n", port)
		http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			// log.Println("--->", req.RemoteAddr, req.URL.String())
		})
		http.ListenAndServe(":"+strconv.Itoa(port), nil)
	},
}

func init() {
	RootCmd.AddCommand(httpservCmd)
}
