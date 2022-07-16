/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"
    "fmt"

    "github.com/aedobrynin/gorsky/util"
	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gorsky",
	Short: "The program makes colored photo from S.M. Prokudin-Gorsky's negatives.",
	Long: "The program makes colored photo from S.M. Prokudin-Gorsky's negatives.",
    CompletionOptions: cobra.CompletionOptions{
        DisableDefaultCmd: true,
    },
    Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("root called")
        util.JustWorks()
    },
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gorsky.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}


