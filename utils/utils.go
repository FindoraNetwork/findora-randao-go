package randao

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

func WaitBlocks(cli *ethclient.Client, bnum uint64) error {
	for {
		curr_block_num, err := cli.BlockNumber(context.Background())
		if err != nil {
			return errors.Wrap(err, "cli.BlockNumber error:")
		}
		if curr_block_num >= bnum {
			break
		}
		// fmt.Println("curr block num: ", curr_block_num)
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

func StoreCampaignId(campaign_ids_path string, campaign_id string) {
	var campaign_ids_path2 = campaign_ids_path
	campaign_ids_path2 = filepath.Join(campaign_ids_path2, "uuid.txt")
	file, err := os.OpenFile(campaign_ids_path2, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("StoreCampaignId err1: %s", err.Error()))
	}
	defer file.Close()

	if _, err := file.Write([]byte(fmt.Sprintf("%s\n", campaign_id))); err != nil {
		panic(fmt.Sprintf("StoreCampaignId err2: %s", err.Error()))
	}

	err = file.Sync()
	if err != nil {
		panic(fmt.Sprintf("StoreCampaignId err3: %s", err.Error()))
	}
}

func RemoveCampaignId(campaign_ids_path string, campaign_id string) {
	campaign_ids1, _ := ReadCampaignIds(campaign_ids_path)
	delete(campaign_ids1, campaign_id)

	DeleteAllCampaignIds(campaign_ids_path)

	for campaign_id := range campaign_ids1 {
		StoreCampaignId(campaign_ids_path, campaign_id)
	}

}

func ReadCampaignIds(campaign_ids_path string) (map[string]struct{}, []string) {
	var campaign_ids_path2 = campaign_ids_path
	campaign_ids_path2 = filepath.Join(campaign_ids_path2, "uuid.txt")
	file, err := os.Open(campaign_ids_path2)
	if err != nil {
		panic(fmt.Sprintf("ReadCampaignIds err1: %s", err.Error()))
	}
	defer file.Close()

	var campaign_ids1 = make(map[string]struct{})
	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		campaign_ids1[string(line)] = struct{}{}
	}

	var campaign_ids2 []string
	for campaign_id := range campaign_ids1 {
		campaign_ids2 = append(campaign_ids2, campaign_id)
	}

	return campaign_ids1, campaign_ids2

}

func DeleteAllCampaignIds(campaign_ids_path string) {
	var campaign_ids_path2 = campaign_ids_path
	campaign_ids_path2 = filepath.Join(campaign_ids_path2, "uuid.txt")
	file, err := os.OpenFile(campaign_ids_path2, os.O_RDWR, 0644)
	if err != nil {
		panic(fmt.Sprintf("DeleteAllCampaignIds err1: %s", err.Error()))
	}
	defer file.Close()

	err = file.Truncate(0)
	if err != nil {
		panic(fmt.Sprintf("DeleteAllCampaignIds err2: %s", err.Error()))
	}

	err = file.Sync()
	if err != nil {
		panic(fmt.Sprintf("DeleteAllCampaignIds err3: %s", err.Error()))
	}
}

func PrintCampaignIds1(campaign_ids map[string]struct{}) {
	for campaign_id := range campaign_ids {
		fmt.Println(campaign_id)
	}
}

func PrintCampaignIds2(campaign_ids []string) {
	for _, campaign_id := range campaign_ids {
		fmt.Println(campaign_id)
	}
}
