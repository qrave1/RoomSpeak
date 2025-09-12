package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "roomspeak",
	Short: "RoomSpeak is a simple audio conferencing application.",
	Run: func(cmd *cobra.Command, args []string) {
		runApp()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
