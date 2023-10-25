package randao_test

import (
	"context"
	utils "findora/randao/utils"
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
)

func init() {
	os.Chdir("..")
}

func TestStoreCampaignId(t *testing.T) {
	var campaign_ids_path = "./campaigns"
	utils.StoreCampaignId(campaign_ids_path, "111111")
	utils.StoreCampaignId(campaign_ids_path, "222222")
	var campaign_ids1, campaign_ids2, _ = utils.ReadCampaignIds(campaign_ids_path)
	printCampaignIds(campaign_ids1, campaign_ids2)

	utils.RemoveCampaignId(campaign_ids_path, "111111")
	campaign_ids1, campaign_ids2, _ = utils.ReadCampaignIds(campaign_ids_path)
	printCampaignIds(campaign_ids1, campaign_ids2)

	utils.StoreCampaignId(campaign_ids_path, "333333")
	campaign_ids1, campaign_ids2, _ = utils.ReadCampaignIds(campaign_ids_path)
	printCampaignIds(campaign_ids1, campaign_ids2)

	utils.DeleteAllCampaignIds(campaign_ids_path)
	campaign_ids1, campaign_ids2, _ = utils.ReadCampaignIds(campaign_ids_path)
	printCampaignIds(campaign_ids1, campaign_ids2)
}

func printCampaignIds(campaign_ids1 map[string]struct{}, campaign_ids2 []string) {
	fmt.Println("----------")
	utils.PrintCampaignIds1(campaign_ids1)
	fmt.Println("**********")
	utils.PrintCampaignIds2(campaign_ids2)
	fmt.Println("----------")
}

func TestWaitBlocks(t *testing.T) {
	cli, err := ethclient.Dial("http://127.0.0.1:8545")
	if err != nil {
		panic(fmt.Sprintf("ethclient.Dial error: %s", err.Error()))
	}

	begin_block_num, err := cli.BlockNumber(context.Background())
	if err != nil {
		panic(fmt.Sprintf("cli.BlockNumber error: %s", err.Error()))
	}
	var end_block_num = begin_block_num + 5
	fmt.Println("begin block num: ", begin_block_num)
	fmt.Println("end block num: ", end_block_num)

	utils.WaitBlocks(cli, end_block_num)

	final_block_num, err := cli.BlockNumber(context.Background())
	if err != nil {
		panic(fmt.Sprintf("cli.BlockNumber error: %s", err.Error()))
	}
	fmt.Println("final block num: ", final_block_num)
}
