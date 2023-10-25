package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var CMDParser = &cobra.Command{
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

type CMDOpt struct {
	Config        string
	CampaignsPath string
}

var CmdOpt1 CMDOpt

func CMDParse() (cmdOpt CMDOpt, err error) {
	CMDParser.Flags().StringVarP(&CmdOpt1.Config, "config", "c", "config.json", "config file path")
	CMDParser.Flags().StringVarP(&CmdOpt1.CampaignsPath, "campaigns", "a", "campaigns", "campaigns directory path")

	err = CMDParser.Execute()
	if err != nil {
		err = errors.Wrap(err, "command parameters error!!!")
		return
	}

	return CmdOpt1, nil
}
