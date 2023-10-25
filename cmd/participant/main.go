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
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

func main() {
	if _, err := CMDParse(); err != nil {
		panic(fmt.Sprintf("command parse error: %s\n", err.Error()))
	}

	configPath, err := filepath.Abs(CmdOpt1.Config)
	if err != nil {
		panic(fmt.Sprintf("filepath.Abs error: %s\n", err.Error()))
	}

	conf_str, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("config file read error: %s\n", err.Error()))
	}

	var conf model.Config
	if err = json.Unmarshal(conf_str, &conf); err != nil {
		panic(fmt.Sprintf("config file parse error: %s\n", err.Error()))
	}
	fmt.Println("config: ", conf)

	if err := os.MkdirAll(CmdOpt1.CampaignsPath, 0755); err != nil {
		panic(fmt.Sprintf("CampaignsPath open or create failed: %s\n", err.Error()))
	}

	_, err = os.Stat(filepath.Join(CmdOpt1.CampaignsPath, "uuid.txt"))
	if os.IsNotExist(err) {
		file, err := os.Create(filepath.Join(CmdOpt1.CampaignsPath, "uuid.txt"))
		if err != nil {
			panic(fmt.Sprintf("Campaigns file create failed: %s\n", err.Error()))
		}
		file.Close()
	} else if err == nil {
	} else {
		panic(fmt.Sprintf("Campaigns file os.Stat error: %s\n", err.Error()))
	}

	cli, err := ethclient.Dial(conf.Chain.Endpoint)
	if err != nil {
		panic(fmt.Sprintf("ethclient.Dial error: %s\n", err.Error()))
	}
	randao1, err := randao.NewRandao(common.HexToAddress(conf.Chain.Randao), cli)
	if err != nil {
		panic(fmt.Sprintf("NewRandao error: %s\n", err.Error()))
	}
	privateKeyECDSA, err := crypto.HexToECDSA(conf.Chain.Campaigner)
	if err != nil {
		panic(fmt.Sprintf("crypto.HexToECDSA( error: %s\n", err.Error()))
	}
	chainID, err := cli.ChainID(context.Background())
	if err != nil {
		panic(fmt.Sprintf("cli.ChainID error: %s\n", err.Error()))
	}

	CampaignIdsUpdateFromChain(randao1)

	_, campaignIds, err := utils.ReadCampaignIds(CmdOpt1.CampaignsPath)
	if err != nil {
		panic(fmt.Sprintf("ReadCampaignIds error: %s\n", err.Error()))
	}
	utils.PrintCampaignIds2(campaignIds)

	var maxTaskCnt = conf.Chain.Opts.MaxCampaigns
	var currTaskCnt uint64 = 0
	var subTaskRets chan *TaskResult = make(chan *TaskResult, maxTaskCnt*100)

	var maxCampaignId, isValid = big.NewInt(0).SetString("-1", 10)
	if !isValid {
		panic("maxCampaignId error!!!")
	}

	for {
		var campaignId *big.Int
		var isNewCampagin = true
		campaignId, isNewCampagin, err = getCampaignId(&campaignIds, CmdOpt1.CampaignsPath, randao1)
		if err != nil {
			fmt.Println("getCampaignId error: ", err)
			handleTaskResult(subTaskRets, &currTaskCnt, maxTaskCnt)
			continue
		}

		var taskStatus *TaskStatus
		var err error
		if isNewCampagin {
			if campaignId.Cmp(maxCampaignId) == 1 {
				taskStatus, err = getTaskStatusFromChain(campaignId, randao1)
				maxCampaignId = campaignId
			} else {
				handleTaskResult(subTaskRets, &currTaskCnt, maxTaskCnt)
				// fmt.Println("campaginId have already be used!!!")
				continue
			}
		} else {
			if campaignId.Cmp(maxCampaignId) == 1 {
				taskStatus, err = getTaskStatusFromFile(CmdOpt1.CampaignsPath, campaignId)
				maxCampaignId = campaignId
			} else {
				handleTaskResult(subTaskRets, &currTaskCnt, maxTaskCnt)
				// fmt.Println("campaginId have already be used!!!")
				continue
			}
		}
		if err != nil {
			fmt.Printf("getTaskStatus error: %s\n", err.Error())
			handleTaskResult(subTaskRets, &currTaskCnt, maxTaskCnt)
			continue
		}
		fmt.Println("campaignId:", campaignId,
			"isNewCampagin:", isNewCampagin,
			"taskStatus:", *taskStatus)

		var workTask *WorkTask = NewWorkTask(taskStatus,
			randao1,
			cli,
			privateKeyECDSA,
			chainID,
			isNewCampagin)

		err = utils.StoreCampaignId(CmdOpt1.CampaignsPath, campaignId.String())
		if err == nil {
			go workTask.DoTask(subTaskRets)
			currTaskCnt++
		} else {
			fmt.Printf("main StoreCampaignId err: %s\n", err.Error())
		}
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
			} else {
				fmt.Println("participate success1, currTaskCnt:", *currTaskCnt,
					"campaignId:", ret1.campaignId)
			}

			utils.RemoveCampaignId(CmdOpt1.CampaignsPath, ret1.campaignId.String())
			RemovesTaskStatusFile(CmdOpt1.CampaignsPath, ret1.campaignId.String())
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
					} else {
						fmt.Println("participate success2, currTaskCnt:", *currTaskCnt,
							"campaignId:", ret1.campaignId,
						)
					}

					utils.RemoveCampaignId(CmdOpt1.CampaignsPath, ret1.campaignId.String())
					RemovesTaskStatusFile(CmdOpt1.CampaignsPath, ret1.campaignId.String())
					(*currTaskCnt)--
				default:
					// fmt.Println("inner layer default loop currTaskCnt: ", *currTaskCnt)
					time.Sleep(time.Millisecond * 1000)
				}
			}
		}
	}
}

func getCampaignId(campaignIds *[]string, campignsPath string, randao1 *randao.Randao) (campaignId *big.Int, isNewCampagin bool, err error) {
	var campaignId_s string
	// time.Sleep(time.Second * 2)
	if len(*campaignIds) != 0 {
		campaignId_s = (*campaignIds)[0]
		var isValid bool
		campaignId, isValid = big.NewInt(0).SetString(campaignId_s, 10)
		(*campaignIds) = (*campaignIds)[1:]

		if !isValid {
			utils.RemoveCampaignId(campignsPath, campaignId_s)
			RemovesTaskStatusFile(campignsPath, campaignId_s)
			err = errors.New("campaignId format is error!!!")
			return
		}
		isNewCampagin = false
	} else {
		var numCampaigns *big.Int
		numCampaigns, err = randao1.NumCampaigns(&bind.CallOpts{
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

func getTaskStatusFromChain(campaignId *big.Int, randao1 *randao.Randao) (taskstatus *TaskStatus, err error) {
	var ret = &TaskStatus{
		CampaignId:   "",
		Step:         0,
		Hs:           "",
		S:            "",
		TxHash:       "",
		RandaoNum:    "",
		CampaignInfo: model.CampaignInfo{},
	}
	_campaignInfo, err := randao1.GetCampaign(&bind.CallOpts{
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

	ret.CampaignInfo = *campaignInfo
	ret.CampaignId = campaignId.String()
	return ret, nil
}

func getTaskStatusFromFile(campignsPath string, campaignId *big.Int) (taskstatus *TaskStatus, err error) {
	return ReadTaskStatusFile(campignsPath, campaignId.String())
}
