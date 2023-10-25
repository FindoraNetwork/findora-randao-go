package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var CMDParser = &cobra.Command{
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

type CmdOpt struct {
	Config string
}

var CmdOpt1 CmdOpt

func CMDParse() (cmdOpt CmdOpt, err error) {
	CMDParser.Flags().StringVarP(&CmdOpt1.Config, "config", "c", "config.json", "config file path")

	err = CMDParser.Execute()
	if err != nil {
		err = errors.Wrap(err, "command parameters error!!!")
		return
	}

	return CmdOpt1, nil
}
