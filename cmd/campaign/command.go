package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "campaigns",
	Example: "campaigns",
	Short:   "randao campaigns client.",
	Long:    `randao campaigns implement by go language.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("campaign running")
		config, err := cmd.Flags().GetString("config")
		if err != nil {
			fmt.Println("parameter config path error!!!")
			return
		}

		fmt.Println("parameter config path is", config)

	},
}

var Config string

func Command() {
	Cmd.Flags().StringVarP(&Config, "config", "c", "config.json", "config file path")

	err := Cmd.Execute()
	if err != nil {
		fmt.Println("command parameters error!!!")
	}
}
