package cmd

import (
	"github.com/spf13/cobra"
	"net/http"
	"strconv"
	"fmt"
)

var httpservCmd = &cobra.Command{
	Use:   "httpserv",
	Short: "Simple HTTP server for testing",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("HTTP server is listening on port %d\n", port)
		http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			println("--->", port, req.URL.String())
		})
		http.ListenAndServe(":" + strconv.Itoa(port), nil)
	},
}

func init() {
	RootCmd.AddCommand(httpservCmd)
}
