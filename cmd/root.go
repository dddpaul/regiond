package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var port int
var metricsPort int

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "regiond",
	Short: "Region based HTTP proxy",
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.regiond.yaml)")
	RootCmd.PersistentFlags().IntVarP(&port, "port", "p", 9090, "port on which the server will listen")
	RootCmd.PersistentFlags().IntVarP(&metricsPort, "metrics-port", "m", 0, "port on which metrics will be exposed")
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".regiond") // name of config file (without extension)
	viper.AddConfigPath("$HOME")    // adding home directory as first search path
	viper.AutomaticEnv()            // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
