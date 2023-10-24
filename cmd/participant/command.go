package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "participant",
	Example: "participant",
	Short:   "randao participant client.",
	Long:    `randao participant implement by go language.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("campaign running")
		config, err := cmd.Flags().GetString("config")
		if err != nil {
			fmt.Println("parameter config path error!!!")
			return
		}

		campaigns, err := cmd.Flags().GetString("campaigns")
		if err != nil {
			fmt.Println("parameter campaigns error!!!")
			return
		}

		fmt.Println("parameter config path is", config)
		fmt.Println("parameter campaigns path is", campaigns)
	},
}

var Config string
var Campaigns string

func Command() {
	Cmd.Flags().StringVarP(&Config, "config", "c", "config.json", "config file path")
	Cmd.Flags().StringVarP(&Campaigns, "campaigns", "a", "campaigns/", "campaigns directory path")

	err := Cmd.Execute()
	if err != nil {
		fmt.Println("command parameters error!!!")
	}
}
