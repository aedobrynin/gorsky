/*
Copyright © 2022 Anton Dobrynin <dobrynin-ae@yandex.ru>
*/

package cmd

import (
	"os"
    "errors"

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
	Long: `The program makes colored photo from S.M. Prokudin-Gorsky's negatives.
It supports .jpeg, .png and .tiff image formats.`,
    CompletionOptions: cobra.CompletionOptions{
        DisableDefaultCmd: true,
    },
    Args: func(cmd *cobra.Command, args []string) error {
        if len(args) < 1 {
            return errors.New("missing filename(s)\nTry " + cmd.CalledAs() + " --help for more information")
        }
        return nil
    },
    Example: `  gorsky image.tif
  gorsky image.png --outdir processed_images`,
    DisableFlagsInUseLine: true,
    SilenceUsage: true,
    RunE: func(cmd *cobra.Command, args []string) error {
        err := util.ProcessImages(args, outDir)
        return err
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
