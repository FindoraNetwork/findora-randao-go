package main

import (
	"context"
	"encoding/json"
	randao "findora/randao/contract"
	model "findora/randao/model"
	utils "findora/randao/utils"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

func main() {
	Command()

	conf_str, err := os.ReadFile(Config)
	if err != nil {
		fmt.Println("config file read error: ", err.Error())
		panic(err)
	}
	var conf model.Config
	err = json.Unmarshal(conf_str, &conf)
	if err != nil {
		fmt.Println("config file parse error: ", err.Error())
		panic(err)
	}
	fmt.Println("config: ", conf)

	cli, err := ethclient.Dial(conf.Chain.Endpoint)
	if err != nil {
		panic(fmt.Sprintf("ethclient.Dial error: %s", err.Error()))
	}
	randao, err := randao.NewRandao(common.HexToAddress(conf.Chain.Randao), cli)
	if err != nil {
		panic(fmt.Sprintf("NewRandao error: %s", err.Error()))
	}
	privateKeyECDSA, err := crypto.HexToECDSA(conf.Chain.Campaigner)
	if err != nil {
		panic(fmt.Sprintf("crypto.HexToECDSA( error: %s", err.Error()))
	}
	chainID, err := cli.ChainID(context.Background())
	if err != nil {
		panic(fmt.Sprintf("cli.ChainID error: %s", err.Error()))
	}

	var _, campaignIds = utils.ReadCampaignIds(conf.CampaginIdsPath)
	utils.PrintCampaignIds2(campaignIds)

	var maxTaskCnt = conf.Chain.Opts.MaxCampaigns
	var currTaskCnt uint64 = 0
	var subTaskRets chan *TaskResult = make(chan *TaskResult, maxTaskCnt*100)

	var maxCampaignId, isValid = big.NewInt(0).SetString("-1", 10)
	if !isValid {
		panic("maxCampaignId error!!!")
	}

	for {
		time.Sleep(time.Millisecond * 1000)
		var campaignId *big.Int
		var isNewCampagin = true
		campaignId, isNewCampagin, err = getCampaignId(&campaignIds, Campaigns, randao)
		if err != nil {
			fmt.Println("getCampaignId error: ", err)
			continue
		}

		var taskStatus *TaskStatus
		var err error
		if isNewCampagin {
			if campaignId.Cmp(maxCampaignId) == 1 {
				taskStatus, err = getTaskStatusFromChain(campaignId, randao)
				maxCampaignId = campaignId
			} else {
				// fmt.Println("campaginId have already be used!!!")
				continue
			}
		} else {
			taskStatus, err = getTaskStatusFromFile(campaignId, Campaigns)
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("campaignId:", campaignId,
			"isNewCampagin:", isNewCampagin,
			"taskStatus:", *taskStatus)

		var workTask *WorkTask = NewWorkTask(taskStatus,
			randao,
			cli,
			privateKeyECDSA,
			chainID,
			isNewCampagin)
		go workTask.DoTask(subTaskRets)

		currTaskCnt++
		handleTaskResult(subTaskRets, &currTaskCnt, maxTaskCnt)
	}

}

func handleTaskResult(subTaskRets chan *TaskResult, currTaskCnt *uint64, maxTaskCnt uint64) {
	var ret1 *TaskResult = nil
	// read once subtask result.
	for ret1 == nil {
		select {
		case ret1 = <-subTaskRets:
			if ret1.err != nil {
				fmt.Println("participate error1, currTaskCnt:", *currTaskCnt,
					"campaignId:", ret1.campaignId,
					"err:", ret1.err)
			}
			if !ret1.isNewCampagin {
				utils.RemoveCampaignId(Campaigns, ret1.campaignId.String())
			}
			fmt.Println("participate success1, currTaskCnt:", *currTaskCnt,
				"campaignId:", ret1.campaignId)
			(*currTaskCnt)--
		default:
			// read subtask result until currTaskCnt < maxTaskCnt.
			ret1 = &TaskResult{}
			for (*currTaskCnt) >= maxTaskCnt {
				select {
				case ret1 = <-subTaskRets:
					if ret1.err != nil {
						fmt.Println("participate error2, currTaskCnt:", *currTaskCnt,
							"campaignId:", ret1.campaignId,
							"err:", ret1.err)
					}
					if !ret1.isNewCampagin {
						utils.RemoveCampaignId(Campaigns, ret1.campaignId.String())
					}
					(*currTaskCnt)--
					fmt.Println("participate success2, currTaskCnt:", *currTaskCnt,
						"campaignId:", ret1.campaignId,
					)
				default:
					// fmt.Println("inner layer default loop currTaskCnt: ", *currTaskCnt)
					time.Sleep(time.Second * 1)
				}
			}
		}
	}
}

func getCampaignId(campaignIds *[]string, campignsPath string, randao *randao.Randao) (campaignId *big.Int, isNewCampagin bool, err error) {
	var campaignId_s string
	// time.Sleep(time.Second * 2)
	if len(*campaignIds) != 0 {
		campaignId_s = (*campaignIds)[0]
		var isValid bool
		campaignId, isValid = big.NewInt(0).SetString(campaignId_s, 10)
		(*campaignIds) = (*campaignIds)[1:]

		if !isValid {
			utils.RemoveCampaignId(campignsPath, campaignId_s)
			err = errors.New("campaignId format is error!!!")
			return
		}
		isNewCampagin = false
	} else {
		var numCampaigns *big.Int
		numCampaigns, err = randao.NumCampaigns(&bind.CallOpts{
			Pending:     false,
			From:        common.Address{},
			BlockNumber: nil,
			Context:     nil})

		if err != nil {
			err = errors.Wrap(err, "NumCampaigns error!!!")
			return
		}
		campaignId = numCampaigns.Sub(numCampaigns, big.NewInt(1))
		isNewCampagin = true
	}
	err = nil
	return
}

func getTaskStatusFromChain(campaignId *big.Int, randao *randao.Randao) (taskstatus *TaskStatus, err error) {
	var ret = &TaskStatus{
		campaign_id: "",
		step:        0,
		hs:          "",
		s:           "",
		tx_hash:     "",
		randao_num:  "",
		campaign_info: model.CampaignInfo{
			Bnum:           "",
			Deposit:        "",
			CommitBalkline: 0,
			CommitDeadline: 0,
			Random:         "",
			Settled:        false,
			Bountypot:      "",
			CommitNum:      0,
			RevealsNum:     0,
		},
	}
	_campaignInfo, err := randao.GetCampaign(&bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: nil,
		Context:     nil},
		campaignId)
	if err != nil {
		err = errors.Wrap(err, "GetCampaign error!!!")
		return
	}

	var campaignInfo = model.CampaignInfoConvert(&_campaignInfo)

	fmt.Println("campaignInfo: ", campaignInfo)

	ret.campaign_info = *campaignInfo
	ret.campaign_id = campaignId.String()
	return ret, nil
}

func getTaskStatusFromFile(campaignId *big.Int, campignsPath string) (taskstatus *TaskStatus, err error) {
	var ret = &TaskStatus{}

	taskstatus_str, err := os.ReadFile(campignsPath + campaignId.String())
	if err != nil {
		utils.RemoveCampaignId(campignsPath, campaignId.String())
		return nil, errors.Wrap(err, "task status file read error")
	}

	err = json.Unmarshal(taskstatus_str, &ret)
	if err != nil {
		utils.RemoveCampaignId(campignsPath, campaignId.String())
		return nil, errors.Wrap(err, "task status file parse error")
	}

	return ret, nil
}
