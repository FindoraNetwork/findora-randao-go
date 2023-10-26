package randao

import (
	"bufio"
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

func WaitBlocks(cli *ethclient.Client, destBnum uint64) (currBnum uint64, err error) {
	currBnum, err = cli.BlockNumber(context.Background())
	if err != nil {
		err = errors.Wrap(err, "cli.BlockNumber error:")
		return
	}
	for currBnum < destBnum {
		currBnum, err = cli.BlockNumber(context.Background())
		if err != nil {
			err = errors.Wrap(err, "cli.BlockNumber error:")
			return
		}
		// fmt.Println("curr block num: ", currBnum)
		time.Sleep(500 * time.Millisecond)
	}
	return currBnum, nil
}

func StoreCampaignId(campaign_ids_path string, campaign_id string) (err error) {
	var campaign_ids_path2 = campaign_ids_path
	campaign_ids_path2 = filepath.Join(campaign_ids_path2, "uuid.txt")
	file, err := os.OpenFile(campaign_ids_path2, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		err = errors.Wrap(err, "StoreCampaignId err1")
		return
	}
	defer file.Close()

	if _, err := file.Write([]byte(fmt.Sprintf("%s\n", campaign_id))); err != nil {
		err = errors.Wrap(err, "StoreCampaignId err2")
		return err
	}

	err = file.Sync()
	if err != nil {
		err = errors.Wrap(err, "StoreCampaignId err3")
		return
	}
	return nil
}

func RemoveCampaignId(campaign_ids_path string, campaign_id string) (err error) {
	campaign_ids1, _, err := ReadCampaignIds(campaign_ids_path)
	if err != nil {
		err = errors.Wrap(err, "RemoveCampaignId err")
		return
	}
	delete(campaign_ids1, campaign_id)

	err = DeleteAllCampaignIds(campaign_ids_path)
	if err != nil {
		return
	}

	for campaign_id := range campaign_ids1 {
		err = StoreCampaignId(campaign_ids_path, campaign_id)
		if err != nil {
			return
		}
	}
	return nil
}

func ReadCampaignIds(campaign_ids_path string) (campaign_ids1 map[string]struct{}, campaign_ids2 []string, err error) {
	var campaign_ids_path2 = campaign_ids_path
	campaign_ids_path2 = filepath.Join(campaign_ids_path2, "uuid.txt")
	file, err := os.Open(campaign_ids_path2)
	if err != nil {
		err = errors.Wrap(err, "ReadCampaignIds err")
		return
	}
	defer file.Close()
	campaign_ids1 = make(map[string]struct{})
	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		campaign_ids1[string(line)] = struct{}{}
	}

	for campaign_id := range campaign_ids1 {
		campaign_ids2 = append(campaign_ids2, campaign_id)
	}

	sort.Slice(campaign_ids2, func(i, j int) bool {
		var campaign_id1, _ = big.NewInt(0).SetString(campaign_ids2[i], 10)
		var campaign_id2, _ = big.NewInt(0).SetString(campaign_ids2[j], 10)
		return campaign_id1.Cmp(campaign_id2) == -1
	})

	return campaign_ids1, campaign_ids2, nil
}

func DeleteAllCampaignIds(campaign_ids_path string) (err error) {
	var campaign_ids_path2 = campaign_ids_path
	campaign_ids_path2 = filepath.Join(campaign_ids_path2, "uuid.txt")
	file, err := os.OpenFile(campaign_ids_path2, os.O_RDWR, 0644)
	if err != nil {
		err = errors.Wrap(err, "DeleteAllCampaignIds err1")
		return
	}
	defer file.Close()

	err = file.Truncate(0)
	if err != nil {
		err = errors.Wrap(err, "DeleteAllCampaignIds err2")
		return
	}

	err = file.Sync()
	if err != nil {
		err = errors.Wrap(err, "DeleteAllCampaignIds err3")
		return
	}
	return nil
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
