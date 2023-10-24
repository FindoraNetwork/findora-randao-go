package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	randao "findora/randao/contract"
	model "findora/randao/model"
	utils "findora/randao/utils"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

type WorkTask struct {
	taskStatus    *TaskStatus
	randao        *randao.Randao
	cli           *ethclient.Client
	key           *ecdsa.PrivateKey
	chainID       *big.Int
	isNewCampagin bool
}
type TaskStatus struct {
	campaign_id   string             `json:"campaign_id"`
	step          uint8              `json:"step"`
	hs            string             `json:"hs"`
	s             string             `json:"_s"`
	tx_hash       string             `json:"tx_hash"`
	randao_num    string             `json:"randao_num"`
	campaign_info model.CampaignInfo `json:"campaign_info"`
}

type TaskResult struct {
	campaignId    *big.Int
	isNewCampagin bool
	err           error
}

func NewWorkTask(taskStatus *TaskStatus,
	randao *randao.Randao,
	cli *ethclient.Client,
	key *ecdsa.PrivateKey,
	chainID *big.Int,
	isNewCampagin bool) *WorkTask {
	var workthd = WorkTask{taskStatus, randao, cli, key, chainID, isNewCampagin}
	return &workthd
}

var txLock sync.Mutex

func (t *WorkTask) Step1() (err error) {
	_s, err := genRandomU256()
	if err != nil {
		err = errors.Wrap(err, "genRandomU256 error!")
		return
	}
	_hs, err := t.randao.ShaCommit(&bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: nil,
		Context:     nil}, _s)
	if err != nil {
		err = errors.Wrap(err, "ShaCommit error!")
		return
	}
	fmt.Println("ShaCommit success, campaignId:", t.taskStatus.campaign_id, "s:", _s.String(), "hs:", _hs)

	t.taskStatus.s = _s.String()
	t.taskStatus.hs = base64.StdEncoding.EncodeToString(_hs[:])
	t.taskStatus.step = 1
	err = nil
	return
}

func (t *WorkTask) Step2() (err error) {
	__hs, err := base64.StdEncoding.DecodeString(t.taskStatus.hs)
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

	campaignId, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_id, 10)
	if !isValid {
		err = errors.New("campainId format is error!")
		return
	}
	deposit, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_info.Deposit, 10)
	if !isValid {
		err = errors.New("deposit format is error!")
		return
	}

	bnum, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_info.Bnum, 10)
	if !isValid {
		err = errors.New("bnum format is error!")
		return
	}

	var commit_balkline = t.taskStatus.campaign_info.CommitBalkline
	var commit_deadline = t.taskStatus.campaign_info.CommitDeadline

	var balkline = bnum.Uint64() - uint64(commit_balkline)
	var deadline = bnum.Uint64() - uint64(commit_deadline)
	currBnum, err := t.cli.BlockNumber(context.Background())
	if err != nil {
		err = errors.Wrap(err, "BlockNumber error!!!")
		return
	}
	if currBnum > deadline {
		err = errors.New("Too late to commit to campaign!!!")
		return
	}
	err = utils.WaitBlocks(t.cli, balkline)
	if err != nil {
		err = errors.Wrap(err, "WaitBlocks error!!!")
		return
	}
	txOpts, err := bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	if err != nil {
		err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
		return
	}

	txLock.Lock()
	defer func() {
		txLock.Unlock()
	}()
	var pendingNonce uint64
	pendingNonce, err = t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	if err != nil {
		err = errors.Wrap(err, "PendingNonceAt error!!!")
		return
	}
	txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)
	// gasPrice, err := t.cli.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	ret.err = errors.Wrap(err, "SuggestGasPrice error!!!")
	// 	res <- ret
	// 	return
	// }
	// txOpts.GasPrice = gasPrice.Mul(gasPrice, big.NewInt(2))
	txOpts.Value = deposit
	tx, err := t.randao.Commit(txOpts, campaignId, _hs)
	if err != nil {
		err = errors.Wrap(err, "Commit error!!!")
		return
	}

	var txHash = tx.Hash().Hex()
	fmt.Println("Commit join success, campaignId:", t.taskStatus.campaign_id, "tx hash:", txHash)

	t.taskStatus.tx_hash = txHash
	t.taskStatus.step = 2
	err = nil
	return
}

func (t *WorkTask) Step3() (err error) {
	// step 3
	campainId, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_id, 10)
	if !isValid {
		err = errors.New("campainId format is error!")
		return
	}

	_s, isValid := big.NewInt(0).SetString(t.taskStatus.s, 10)
	if !isValid {
		err = errors.New("_s format is error!")
		return
	}

	bnum, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_info.Bnum, 10)
	if !isValid {
		err = errors.New("bnum format is error!")
		return
	}

	var commit_deadline = t.taskStatus.campaign_info.CommitDeadline
	var deadline = bnum.Uint64() - uint64(commit_deadline)

	currBnum, err := t.cli.BlockNumber(context.Background())
	if err != nil {
		err = errors.Wrap(err, "BlockNumber error!!!")
		return
	}
	if currBnum > bnum.Uint64() {
		err = errors.New("Too late to reveal to campaign!!!")
		return
	}
	utils.WaitBlocks(t.cli, deadline)

	var txHash = t.taskStatus.tx_hash
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
	fmt.Println("Commit exec success, campaignId:", t.taskStatus.campaign_id, "tx receipt:", receipt)

	txOpts, err := bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	if err != nil {
		err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
		return
	}

	txLock.Lock()
	defer func() {
		txLock.Unlock()
	}()
	var pendingNonce uint64
	pendingNonce, err = t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	if err != nil {
		err = errors.Wrap(err, "PendingNonceAt error!!!")
		return
	}
	txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)
	// gasPrice, err = t.cli.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	ret.err = errors.Wrap(err, "SuggestGasPrice error!!!")
	// 	res <- ret
	// 	return
	// }
	// txOpts.GasPrice = gasPrice.Mul(gasPrice, big.NewInt(2))
	tx, err = t.randao.Reveal(txOpts, campainId, _s)
	if err != nil {
		err = errors.Wrap(err, "Reveal error!!!")
		return
	}

	txHash = tx.Hash().Hex()
	fmt.Println("Reveal join success, campaignId:", t.taskStatus.campaign_id, "tx hash:", txHash)
	t.taskStatus.tx_hash = txHash
	t.taskStatus.step = 3
	err = nil
	return
}

func (t *WorkTask) Step4() (err error) {
	campainId, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_id, 10)
	if !isValid {
		err = errors.New("campainId format is error!")
		return
	}

	bnum, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_info.Bnum, 10)
	if !isValid {
		err = errors.New("bnum format is error!")
		return
	}

	utils.WaitBlocks(t.cli, bnum.Uint64())

	var txHash = t.taskStatus.tx_hash
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
	fmt.Println("Reveal exec success, campaignId:", t.taskStatus.campaign_id, "tx receipt:", receipt)

	txOpts, err := bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	if err != nil {
		err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
		return
	}

	txLock.Lock()
	var pendingNonce uint64
	pendingNonce, err = t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	if err != nil {
		err = errors.Wrap(err, "PendingNonceAt error!!!")
		txLock.Unlock()
		return
	}
	txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)
	// gasPrice, err = t.cli.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	ret.err = errors.Wrap(err, "SuggestGasPrice error!!!")
	// 	res <- ret
	// 	return
	// }
	// txOpts.GasPrice = gasPrice
	tx, err = t.randao.GetMyBounty(txOpts, campainId)
	if err != nil {
		err = errors.Wrap(err, "GetMyBounty error!!!")
		txLock.Unlock()
		return
	}
	txLock.Unlock()

	txHash = tx.Hash().Hex()
	fmt.Println("GetMyBounty join success, campaignId:", t.taskStatus.campaign_id, "tx hash:", txHash)

	receipt, err = bind.WaitMined(context.Background(), t.cli, tx)
	if err != nil {
		err = errors.Wrap(err, "GetMyBounty WaitMined error!!!")
		return
	}
	if receipt.Status != 1 {
		err = errors.New("GetMyBounty receipt.Status not equal 1!!!")
		return
	}
	fmt.Println("GetMyBounty exec success, campaignId:", t.taskStatus.campaign_id, "tx receipt:", receipt)

	t.taskStatus.tx_hash = txHash
	t.taskStatus.step = 4
	err = nil
	return
}

func (t *WorkTask) DoTask(res chan<- *TaskResult) {
	var ret = &TaskResult{}

	campaignId, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_id, 10)
	if !isValid {
		ret.err = errors.New("campainId format is error!")
		res <- ret
		return
	}
	ret.campaignId = campaignId

	if t.taskStatus.step == 0 {
		err := t.Step1()
		if err != nil {
			ret.err = errors.Wrap(err, "step1 error")
			res <- ret
			return
		}
	}

	if t.taskStatus.step == 1 {
		err := t.Step2()
		if err != nil {
			ret.err = errors.Wrap(err, "step2 error")
			res <- ret
			return
		}
	}

	if t.taskStatus.step == 2 {
		err := t.Step3()
		if err != nil {
			ret.err = errors.Wrap(err, "step3 error")
			res <- ret
			return
		}
	}

	if t.taskStatus.step == 3 {
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
			err = errors.New(fmt.Sprint("rand number error: %d", n))
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
