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
    maxWorkers int
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
        if maxWorkers <= 0 {
            return errors.New("workers count must be positive")
        }
        return nil
    },
    Example: `  gorsky image.tif
  gorsky image.png --outdir processed_images
  gorsky image1.png image2.png image3.png image4.png image5.png --maxworkers 5`,
    DisableFlagsInUseLine: true,
    SilenceUsage: true,
    RunE: func(cmd *cobra.Command, args []string) error {
        err := util.ProcessImages(args, outDir, maxWorkers)
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
    rootCmd.Flags().StringVarP(&outDir, "outdir", "o", "result", "Result images will be stored in this folder")
    rootCmd.Flags().IntVarP(&maxWorkers, "maxworkers", "m", 4, "How many images can be processed simultaneously")
}
