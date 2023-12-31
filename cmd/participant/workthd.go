package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	randao "findora/randao/contract"
	model "findora/randao/model"
	utils "findora/randao/utils"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

type WorkTask struct {
	taskStatus    *TaskStatus
	randao1       *randao.Randao
	cli           *ethclient.Client
	key           *ecdsa.PrivateKey
	chainID       *big.Int
	isNewCampagin bool
}
type TaskStatus struct {
	CampaignId   string             `json:"campaign_id"`
	Step         uint8              `json:"step"`
	Hs           string             `json:"hs"`
	S            string             `json:"_s"`
	TxHash       string             `json:"tx_hash"`
	RandaoNum    string             `json:"randao_num"`
	CampaignInfo model.CampaignInfo `json:"campaign_info"`
}

type TaskResult struct {
	campaignId    *big.Int
	isNewCampagin bool
	err           error
}

func NewWorkTask(taskStatus *TaskStatus,
	randao1 *randao.Randao,
	cli *ethclient.Client,
	key *ecdsa.PrivateKey,
	chainID *big.Int,
	isNewCampagin bool) *WorkTask {
	var workthd = WorkTask{taskStatus, randao1, cli, key, chainID, isNewCampagin}
	return &workthd
}

var txLock sync.Mutex

func (t *WorkTask) Step1() (err error) {
	_s, err := genRandomU256()
	if err != nil {
		err = errors.Wrap(err, "genRandomU256 error!")
		return
	}
	_hs, err := t.randao1.ShaCommit(&bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: nil,
		Context:     nil}, _s)
	if err != nil {
		err = errors.Wrap(err, "ShaCommit error!")
		return
	}
	fmt.Println("ShaCommit success, campaignId:", t.taskStatus.CampaignId, ", s:", _s.String(), ", hs:", _hs)

	t.taskStatus.Step = 1
	t.taskStatus.S = _s.String()
	t.taskStatus.Hs = base64.StdEncoding.EncodeToString(_hs[:])

	err = StoreTaskStatusFile(CmdOpt1.CampaignsPath, t.taskStatus)
	if err != nil {
		err = errors.Wrap(err, "storeTaskStatusToFile err")
		return
	}
	err = nil
	return
}

func (t *WorkTask) Step2() (err error) {
	__hs, err := base64.StdEncoding.DecodeString(t.taskStatus.Hs)
	if err != nil {
		err = errors.New("hs string base64 decode error!")
		return
	}
	if len(__hs) != 32 {
		err = errors.New("hs string base64 decode length error!")
		return
	}
	var _hs [32]byte
	copy(_hs[:], __hs)

	campaignId, isValid := big.NewInt(0).SetString(t.taskStatus.CampaignId, 10)
	if !isValid {
		err = errors.New("campainId format is error!")
		return
	}
	deposit, isValid := big.NewInt(0).SetString(t.taskStatus.CampaignInfo.Deposit, 10)
	if !isValid {
		err = errors.New("deposit format is error!")
		return
	}

	bnum, isValid := big.NewInt(0).SetString(t.taskStatus.CampaignInfo.Bnum, 10)
	if !isValid {
		err = errors.New("bnum format is error!")
		return
	}

	var commit_balkline = t.taskStatus.CampaignInfo.CommitBalkline
	var commit_deadline = t.taskStatus.CampaignInfo.CommitDeadline

	var balkline = bnum.Uint64() - uint64(commit_balkline)
	var deadline = bnum.Uint64() - uint64(commit_deadline)

	currBnum, err := utils.WaitBlocks(t.cli, balkline)
	if err != nil {
		err = errors.Wrap(err, "WaitBlocks error!!!")
		return
	}
	if currBnum > deadline {
		err = errors.New("Too late to commit to campaign!!!")
		return
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	if err != nil {
		err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
		return
	}

	txLock.Lock()
	fmt.Println("Step2 lock campaignId:", campaignId)
	defer func() {
		txLock.Unlock()
		fmt.Println("Step2 unlock campaignId:", campaignId)
	}()

	// randaoAbi, err := abi.JSON(strings.NewReader(contract.RandaoABI))
	// if err != nil {
	// 	err = errors.Wrap(err, "abi.JSON error")
	// 	return
	// }
	// data, err := randaoAbi.Pack("commit", campaignId, _hs)
	// if err != nil {
	// 	err = errors.Wrap(err, "randaoAbi.Pack error")
	// 	return
	// }
	// to := common.HexToAddress(model.Conf.Chain.Randao)
	// estimateGasCall := ethereum.CallMsg{
	// 	To:    &to,
	// 	Data:  data,
	// 	Value: deposit,
	// }
	// gasLimit, err := t.cli.EstimateGas(context.Background(), estimateGasCall)
	// if err != nil {
	// 	err = errors.Wrap(err, "EstimateGas error")
	// 	return
	// }
	// gasPrice, err := t.cli.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	err = errors.Wrap(err, "cli.SuggestGasPrice error")
	// 	return
	// }
	// gasPrice = gasPrice.Mul(gasPrice, big.NewInt(10))
	// gasLimit = gasLimit * 100

	var pendingNonce uint64
	pendingNonce, err = t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	if err != nil {
		err = errors.Wrap(err, "PendingNonceAt error!!!")
		return
	}
	// txOpts.GasPrice = gasPrice
	// txOpts.GasLimit = gasLimit
	txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)
	txOpts.Value = deposit

	fmt.Println("Step2 campaignId:", campaignId, "pendingNonce:", pendingNonce)
	tx, err := t.randao1.Commit(txOpts, campaignId, _hs)
	if err != nil {
		err = errors.Wrap(err, "Commit error!!!")
		return
	}

	var txHash = tx.Hash().Hex()
	fmt.Println("Commit join success, campaignId:", t.taskStatus.CampaignId, "tx hash:", txHash)

	t.taskStatus.Step = 2
	t.taskStatus.TxHash = txHash

	err = StoreTaskStatusFile(CmdOpt1.CampaignsPath, t.taskStatus)
	if err != nil {
		err = errors.Wrap(err, "storeTaskStatusToFile err")
		return
	}
	err = nil
	return
}

func (t *WorkTask) Step3() (err error) {
	// step 3
	campaignId, isValid := big.NewInt(0).SetString(t.taskStatus.CampaignId, 10)
	if !isValid {
		err = errors.New("campainId format is error!")
		return
	}

	_s, isValid := big.NewInt(0).SetString(t.taskStatus.S, 10)
	if !isValid {
		err = errors.New("_s format is error!")
		return
	}

	bnum, isValid := big.NewInt(0).SetString(t.taskStatus.CampaignInfo.Bnum, 10)
	if !isValid {
		err = errors.New("bnum format is error!")
		return
	}

	var commit_deadline = t.taskStatus.CampaignInfo.CommitDeadline
	var deadline = bnum.Uint64() - uint64(commit_deadline)

	currBnum, err := utils.WaitBlocks(t.cli, deadline+1)
	if err != nil {
		err = errors.Wrap(err, "WaitBlocks error!!!")
		return
	}
	if currBnum >= bnum.Uint64() {
		err = errors.New("Too late to reveal to campaign!!!")
		return
	}

	var txHash = t.taskStatus.TxHash
	tx, _, err := t.cli.TransactionByHash(context.Background(), common.HexToHash(txHash))
	if err != nil {
		err = errors.Wrap(err, "TransactionByHash error!!!")
		return
	}

	receipt, err := bind.WaitMined(context.Background(), t.cli, tx)
	if err != nil {
		err = errors.Wrap(err, "Commit WaitMined error!!!")
		return
	}
	if receipt.Status != 1 {
		err = errors.New("Commit receipt.Status not equal 1!!!")
		return
	}
	fmt.Println("Commit exec success, campaignId:", t.taskStatus.CampaignId, "tx receipt:", receipt)

	txOpts, err := bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	if err != nil {
		err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
		return
	}

	txLock.Lock()
	fmt.Println("Step3 lock campaignId:", campaignId)
	defer func() {
		txLock.Unlock()
		fmt.Println("Step3 unlock campaignId:", campaignId)
	}()

	// randaoAbi, err := abi.JSON(strings.NewReader(contract.RandaoABI))
	// if err != nil {
	// 	err = errors.Wrap(err, "abi.JSON error")
	// 	return
	// }
	// fmt.Println("reveal", campaignId, _s)

	// data, err := randaoAbi.Pack("reveal", campaignId, _s)
	// if err != nil {
	// 	err = errors.Wrap(err, "randaoAbi.Pack error")
	// 	return
	// }
	// to := common.HexToAddress(model.Conf.Chain.Randao)
	// estimateGasCall := ethereum.CallMsg{
	// 	To:   &to,
	// 	Data: data,
	// }
	// gasLimit, err := t.cli.EstimateGas(context.Background(), estimateGasCall)
	// if err != nil {
	// 	err = errors.Wrap(err, "EstimateGas error")
	// 	return
	// }
	// gasPrice, err := t.cli.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	err = errors.Wrap(err, "cli.SuggestGasPrice error")
	// 	return
	// }
	// gasPrice = gasPrice.Mul(gasPrice, big.NewInt(3))
	// gasLimit = gasLimit * 5

	var pendingNonce uint64
	pendingNonce, err = t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	if err != nil {
		err = errors.Wrap(err, "PendingNonceAt error!!!")
		return
	}
	// txOpts.GasPrice = gasPrice
	// txOpts.GasLimit = gasLimit
	txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)

	fmt.Println("Step3 campaignId:", campaignId, "pendingNonce:", pendingNonce)
	tx, err = t.randao1.Reveal(txOpts, campaignId, _s)
	if err != nil {
		err = errors.Wrap(err, "Reveal error!!!")
		return
	}

	txHash = tx.Hash().Hex()
	fmt.Println("Reveal join success, campaignId:", t.taskStatus.CampaignId, "tx hash:", txHash)

	t.taskStatus.Step = 3
	t.taskStatus.TxHash = txHash

	err = StoreTaskStatusFile(CmdOpt1.CampaignsPath, t.taskStatus)
	if err != nil {
		err = errors.Wrap(err, "storeTaskStatusToFile err")
		return
	}
	err = nil
	return
}

func (t *WorkTask) Step4() (err error) {
	campaignId, isValid := big.NewInt(0).SetString(t.taskStatus.CampaignId, 10)
	if !isValid {
		err = errors.New("campainId format is error!")
		return
	}

	bnum, isValid := big.NewInt(0).SetString(t.taskStatus.CampaignInfo.Bnum, 10)
	if !isValid {
		err = errors.New("bnum format is error!")
		return
	}

	_, err = utils.WaitBlocks(t.cli, bnum.Uint64())
	if err != nil {
		err = errors.Wrap(err, "WaitBlocks error!!!")
		return
	}

	var txHash = t.taskStatus.TxHash
	tx, _, err := t.cli.TransactionByHash(context.Background(), common.HexToHash(txHash))
	if err != nil {
		err = errors.Wrap(err, "TransactionByHash error!!!")
		return
	}

	receipt, err := bind.WaitMined(context.Background(), t.cli, tx)
	if err != nil {
		err = errors.Wrap(err, "Reveal WaitMined error!!!")
		return
	}
	if receipt.Status != 1 {
		err = errors.New("Reveal receipt.Status not equal 1!!!")
		return
	}
	fmt.Println("Reveal exec success, campaignId:", t.taskStatus.CampaignId, "tx receipt:", receipt)

	txOpts, err := bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	if err != nil {
		err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
		return
	}

	txLock.Lock()
	fmt.Println("Step4 lock campaignId:", campaignId)

	// randaoAbi, err := abi.JSON(strings.NewReader(contract.RandaoABI))
	// if err != nil {
	// 	err = errors.Wrap(err, "abi.JSON error")
	// 	return
	// }
	// data, err := randaoAbi.Pack("getMyBounty", campaignId)
	// if err != nil {
	// 	err = errors.Wrap(err, "randaoAbi.Pack error")
	// 	return
	// }
	// to := common.HexToAddress(model.Conf.Chain.Randao)
	// estimateGasCall := ethereum.CallMsg{
	// 	To:   &to,
	// 	Data: data,
	// }
	// gasLimit, err := t.cli.EstimateGas(context.Background(), estimateGasCall)
	// if err != nil {
	// 	err = errors.Wrap(err, "EstimateGas error")
	// 	return
	// }
	// gasPrice, err := t.cli.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	err = errors.Wrap(err, "cli.SuggestGasPrice error")
	// 	return
	// }
	// gasPrice = gasPrice.Mul(gasPrice, big.NewInt(3))
	// gasLimit = gasLimit * 5

	var pendingNonce uint64
	pendingNonce, err = t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	if err != nil {
		txLock.Unlock()
		fmt.Println("Step4 unlock campaignId:", campaignId)

		err = errors.Wrap(err, "PendingNonceAt error!!!")
		return
	}
	// txOpts.GasPrice = gasPrice
	// txOpts.GasLimit = gasLimit
	txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)

	fmt.Println("Step4 campaignId:", campaignId, "pendingNonce:", pendingNonce)
	// gasPrice, err = t.cli.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	ret.err = errors.Wrap(err, "SuggestGasPrice error!!!")
	// 	res <- ret
	// 	return
	// }
	// txOpts.GasPrice = gasPrice
	tx, err = t.randao1.GetMyBounty(txOpts, campaignId)
	if err != nil {
		err = errors.Wrap(err, "GetMyBounty error!!!")
		fmt.Println("Step4 unlock campaignId:", campaignId)

		txLock.Unlock()
		return
	}
	txLock.Unlock()
	fmt.Println("Step4 unlock campaignId:", campaignId)

	txHash = tx.Hash().Hex()
	fmt.Println("GetMyBounty join success, campaignId:", t.taskStatus.CampaignId, "tx hash:", txHash)

	receipt, err = bind.WaitMined(context.Background(), t.cli, tx)
	if err != nil {
		err = errors.Wrap(err, "GetMyBounty WaitMined error!!!")
		return
	}
	if receipt.Status != 1 {
		err = errors.New("GetMyBounty receipt.Status not equal 1!!!")
		return
	}
	fmt.Println("GetMyBounty exec success, campaignId:", t.taskStatus.CampaignId, "tx receipt:", receipt)

	t.taskStatus.Step = 4
	t.taskStatus.TxHash = txHash

	err = StoreTaskStatusFile(CmdOpt1.CampaignsPath, t.taskStatus)
	if err != nil {
		err = errors.Wrap(err, "storeTaskStatusToFile err")
		return
	}
	err = nil
	return
}

func (t *WorkTask) DoTask(res chan<- *TaskResult) {
	var ret = &TaskResult{}

	campaignId, isValid := big.NewInt(0).SetString(t.taskStatus.CampaignId, 10)
	if !isValid {
		ret.err = errors.New("campainId format is error!")
		res <- ret
		return
	}
	ret.campaignId = campaignId

	if t.taskStatus.Step == 0 {
		err := t.Step1()
		if err != nil {
			ret.err = errors.Wrap(err, "step1 error")
			res <- ret
			return
		}
	}

	if t.taskStatus.Step == 1 {
		err := t.Step2()
		if err != nil {
			ret.err = errors.Wrap(err, "step2 error")
			res <- ret
			return
		}
	}

	if t.taskStatus.Step == 2 {
		err := t.Step3()
		if err != nil {
			ret.err = errors.Wrap(err, "step3 error")
			res <- ret
			return
		}
	}

	if t.taskStatus.Step == 3 {
		err := t.Step4()
		if err != nil {
			ret.err = errors.Wrap(err, "step4 error")
			res <- ret
			return
		}
	}

	ret.isNewCampagin = t.isNewCampagin
	ret.err = nil
	res <- ret
}

func genRandomU256() (random *big.Int, err error) {
	m := make(map[uint64]struct{})

	for len(m) < 4 {
		var buf [8]byte
		var num uint64
		var n int
		n, err = rand.Read(buf[:])
		if err != nil {
			err = errors.Wrap(err, "gen random failed!!!")
			return
		}
		if n < 8 {
			err = errors.New(fmt.Sprintf("rand number error: %d", n))
			return
		}
		num = uint64(binary.LittleEndian.Uint64(buf[:]))
		m[num] = struct{}{}
	}

	var nums []uint64
	for k := range m {
		nums = append(nums, k)
	}

	random = big.NewInt(0)
	for _, n := range nums {
		random.Lsh(random, 64)
		random.Or(random, big.NewInt(int64(n)))
	}

	return
}

func ReadTaskStatusFile(campignsPath string, campaignId string) (taskStatus *TaskStatus, err error) {
	taskStatus = &TaskStatus{}

	taskstatus_str, err := os.ReadFile(filepath.Join(campignsPath, campaignId))
	if err != nil {
		utils.RemoveCampaignId(campignsPath, campaignId)
		RemovesTaskStatusFile(campignsPath, campaignId)
		return nil, errors.Wrap(err, "ReadTaskStatusFile error1")
	}

	err = json.Unmarshal(taskstatus_str, &taskStatus)
	if err != nil {
		utils.RemoveCampaignId(campignsPath, campaignId)
		RemovesTaskStatusFile(campignsPath, campaignId)
		return nil, errors.Wrap(err, "ReadTaskStatusFile error2")
	}

	return taskStatus, nil
}

func RemovesTaskStatusFile(campignsPath string, campaignId string) (err error) {
	err = os.Remove(filepath.Join(campignsPath, campaignId))
	if err != nil {
		err = errors.Wrap(err, "RemovesTaskStatusFile error")
		return
	}

	return nil
}

func StoreTaskStatusFile(campignsPath string, taskStatus *TaskStatus) (err error) {
	taskStatus_s, err := json.Marshal(*taskStatus)
	if err != nil {
		err = errors.Wrap(err, "StoreTaskStatusFile error1")
		return
	}

	file, err := os.OpenFile(filepath.Join(campignsPath, taskStatus.CampaignId), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		err = errors.Wrap(err, "StoreTaskStatusFile error2")
		return
	}
	defer file.Close()

	_, err = file.Write(taskStatus_s)
	if err != nil {
		err = errors.Wrap(err, "StoreTaskStatusFile error3")
		return
	}

	return nil
}

var (
	CampignIdsFromChain []string
	CampignIdsFromFile  []string
	CampignIdsLock      sync.Mutex
)

var subscribedBlock uint64 = 0

func CampaignIdsUpdateFromChain(evRandao *randao.Randao, cli *ethclient.Client) {
	for {
		curr_block_num, err := cli.BlockNumber(context.Background())
		if err != nil {
			panic(errors.Wrap(err, "cli.BlockNumber error"))
		}

		if curr_block_num <= subscribedBlock {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		subscribedBlock = curr_block_num

		iter, err := evRandao.FilterLogCampaignAdded(
			&bind.FilterOpts{Start: subscribedBlock},
			nil, nil, nil)
		if err != nil {
			panic(fmt.Sprintf("FilterLogCampaignAdded watch create failed: %s\n", err.Error()))
		}

		for iter.Next() {
			if err = iter.Error(); err != nil {
				fmt.Println("LogCampaignAdded iterator error: ", err)
				break
			}
			evt := iter.Event
			fmt.Println("LogCampaignAdded event: ", evt)

			var is_exist = func(s []string, x string) bool {
				for _, v := range s {
					if strings.Compare(v, x) == 0 {
						return true
					}
				}
				return false
			}

			CampignIdsLock.Lock()
			var campaignID = evt.CampaignID.String()
			if !is_exist(CampignIdsFromFile, campaignID) && !is_exist(CampignIdsFromChain, campaignID) {
				CampignIdsFromChain = append(CampignIdsFromChain, campaignID)
				fmt.Println("campaignID insert CampignIdsFromChain: ", campaignID)
			} else {
				fmt.Println("campaignID exist in CampignIdsFromFile or CampignIdsFromChain:", campaignID)
			}
			CampignIdsLock.Unlock()
		}
		iter.Close()
	}
	// for {
	// 	select {
	// 	case err := <-sub.Err():
	// 		panic(fmt.Sprintf("WatchLogCampaignAdded subscribe error: %s\n", err.Error()))
	// 	case evt := <-logCampaignAdded:
	// 		{
	// 			fmt.Printf("event logCampaignAdded: %s\n", evt.CampaignID.String())
	// 			var is_exist = func(s []string, x string) bool {
	// 				for _, v := range s {
	// 					if strings.Compare(v, x) == 0 {
	// 						return true
	// 					}
	// 				}
	// 				return false
	// 			}

	// 			CampignIdsLock.Lock()
	// 			var campaignID = evt.CampaignID.String()
	// 			if !is_exist(CampignIdsFromFile, campaignID) && !is_exist(CampignIdsFromChain, campaignID) {
	// 				CampignIdsFromChain = append(CampignIdsFromChain, campaignID)
	// 				fmt.Println("campaignID insert CampignIdsFromChain: ", campaignID)
	// 			} else {
	// 				fmt.Println("campaignID exist in CampignIdsFromFile or CampignIdsFromChain:", campaignID)
	// 			}
	// 			CampignIdsLock.Unlock()
	// 		}
	// 	}
	// }
}
