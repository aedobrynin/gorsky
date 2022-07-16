/*
Copyright © 2022 Anton Dobrynin <dobrynin-ae@yandex.ru>
*/

package cmd

import (
	"os"
    "fmt"

    "github.com/aedobrynin/gorsky/util"
	"github.com/spf13/cobra"
)

var (
    // Used for flags
    outDir string
)

var rootCmd = &cobra.Command{
	Use:   "gorsky <path_to_negative>",
	Short: "The program makes colored photo from S.M. Prokudin-Gorsky's negatives.",
	Long: "The program makes colored photo from S.M. Prokudin-Gorsky's negatives.",
    CompletionOptions: cobra.CompletionOptions{
        DisableDefaultCmd: true,
    },
    Args: cobra.MinimumNArgs(1),
    Example: `gorsky image.tif
    gorsky image.png --outdir processed_images`,
    DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("root called")
        util.JustWorks()
    },
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
    rootCmd.Flags().StringVar(&outDir, "outdir", "result", "Result images will be stored in this folder")
}
