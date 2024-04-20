/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hexahigh/yapc/cli/lib/config"
)

var (
	// Used for flags.
	cfgFile *string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "yapc-cli",
	Short: "Simple CLI for YAPC",
	Long:  `A CLI for uploading files to YAPC.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	cfgFile = rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default is os.userConfigDir/yapc-cli/config.json)")
}

func initConfig() {
	if *cfgFile != "" {
		config.CheckAndGenerate(*cfgFile)
	} else {
		*cfgFile = config.GetDefaultLocation()
		config.CheckAndGenerate(*cfgFile)
	}

	fmt.Println("Using config file:", *cfgFile)
}
