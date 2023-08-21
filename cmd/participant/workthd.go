package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	randao "findora/randao/contract"
	model "findora/randao/model"
	"fmt"
	"math/big"

	utils "findora/randao/utils"

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
	step          uint8              `json:"step"`
	hs            string             `json:"hs"`
	randao_num    string             `json:"randao_num"`
	s             string             `json:"_s"`
	campaign_id   string             `json:"campaign_id"`
	campaign_info model.CampaignInfo `json:"campaign_info"`
	tx_hash       string             `json:"tx_hash"`
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

func (t *WorkTask) DoTask(res chan<- *TaskResult) {
	var ret = &TaskResult{}
	campainId, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_id, 10)
	if !isValid {
		ret.err = errors.New("campainId format is error!")
		res <- ret
		return
	}

	ret.campaignId = campainId

	deposit, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_info.Deposit, 10)
	if !isValid {
		ret.err = errors.New("deposit format is error!")
		res <- ret
		return
	}

	bnum, isValid := big.NewInt(0).SetString(t.taskStatus.campaign_info.Bnum, 10)
	if !isValid {
		ret.err = errors.New("bnum format is error!")
		res <- ret
		return
	}

	var commit_balkline = t.taskStatus.campaign_info.CommitBalkline
	var commit_deadline = t.taskStatus.campaign_info.CommitDeadline

	// step 1
	_s, err := genRandomU256()
	if err != nil {
		ret.err = errors.Wrap(err, "genRandomU256 error!")
		res <- ret
		return
	}
	_hs, err := t.randao.ShaCommit(&bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: nil,
		Context:     nil}, _s)
	if err != nil {
		ret.err = errors.Wrap(err, "ShaCommit error!")
		res <- ret
		return
	}
	fmt.Println("ShaCommit success: ", _hs)

	// step 2
	var balkline = bnum.Uint64() - uint64(commit_balkline)
	var deadline = bnum.Uint64() - uint64(commit_deadline)
	currBnum, err := t.cli.BlockNumber(context.Background())
	if err != nil {
		ret.err = errors.Wrap(err, "BlockNumber error!!!")
		res <- ret
		return
	}
	if currBnum > deadline {
		ret.err = errors.New("Too late to commit to campaign!!!")
		res <- ret
		return
	}
	err = utils.WaitBlocks(t.cli, balkline)
	if err != nil {
		ret.err = errors.Wrap(err, "WaitBlocks error!!!")
		res <- ret
		return
	}
	txOpts, err := bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	if err != nil {
		ret.err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
		res <- ret
		return
	}
	pendingNonce, err := t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	if err != nil {
		ret.err = errors.Wrap(err, "PendingNonceAt error!!!")
		res <- ret
		return
	}
	txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)
	gasPrice, err := t.cli.SuggestGasPrice(context.Background())
	if err != nil {
		ret.err = errors.Wrap(err, "SuggestGasPrice error!!!")
		res <- ret
		return
	}
	txOpts.GasPrice = gasPrice.Mul(gasPrice, big.NewInt(2))
	txOpts.Value = deposit
	tx, err := t.randao.Commit(txOpts, campainId, _hs)
	if err != nil {
		ret.err = errors.Wrap(err, "Commit error!!!")
		res <- ret
		return
	}
	// fmt.Println("tx hash", tx.Hash())
	receipt, err := bind.WaitMined(context.Background(), t.cli, tx)
	if err != nil {
		ret.err = errors.Wrap(err, "Commit WaitMined error!!!")
		res <- ret
		return
	}
	if receipt.Status != 1 {
		ret.err = errors.New("Commit receipt.Status not equal 1!!!")
		res <- ret
		return
	}
	fmt.Println("Commit success: ", receipt)

	// step 3
	currBnum, err = t.cli.BlockNumber(context.Background())
	if err != nil {
		ret.err = errors.Wrap(err, "BlockNumber error!!!")
		res <- ret
		return
	}
	if currBnum > bnum.Uint64() {
		ret.err = errors.New("Too late to reveal to campaign!!!")
		res <- ret
		return
	}
	utils.WaitBlocks(t.cli, deadline)
	txOpts, err = bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	if err != nil {
		ret.err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
		res <- ret
		return
	}
	pendingNonce, err = t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	if err != nil {
		ret.err = errors.Wrap(err, "PendingNonceAt error!!!")
		res <- ret
		return
	}
	txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)
	gasPrice, err = t.cli.SuggestGasPrice(context.Background())
	if err != nil {
		ret.err = errors.Wrap(err, "SuggestGasPrice error!!!")
		res <- ret
		return
	}
	txOpts.GasPrice = gasPrice.Mul(gasPrice, big.NewInt(2))
	tx, err = t.randao.Reveal(txOpts, campainId, _s)
	if err != nil {
		ret.err = errors.Wrap(err, "Reveal error!!!")
		res <- ret
		return
	}
	// fmt.Println("tx hash", tx.Hash())
	receipt, err = bind.WaitMined(context.Background(), t.cli, tx)
	if err != nil {
		ret.err = errors.Wrap(err, "Reveal WaitMined error!!!")
		res <- ret
		return
	}
	if receipt.Status != 1 {
		ret.err = errors.New("Reveal receipt.Status not equal 1!!!")
		res <- ret
		return
	}
	fmt.Println("Reveal success: ", receipt)

	// step 4
	utils.WaitBlocks(t.cli, bnum.Uint64())
	txOpts, err = bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	if err != nil {
		ret.err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
		res <- ret
		return
	}
	pendingNonce, err = t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	if err != nil {
		ret.err = errors.Wrap(err, "PendingNonceAt error!!!")
		res <- ret
		return
	}
	txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)
	gasPrice, err = t.cli.SuggestGasPrice(context.Background())
	if err != nil {
		ret.err = errors.Wrap(err, "SuggestGasPrice error!!!")
		res <- ret
		return
	}
	txOpts.GasPrice = gasPrice
	tx, err = t.randao.GetMyBounty(txOpts, campainId)
	if err != nil {
		ret.err = errors.Wrap(err, "GetMyBounty error!!!")
		res <- ret
		return
	}
	// fmt.Println("tx hash", tx.Hash())
	receipt, err = bind.WaitMined(context.Background(), t.cli, tx)
	if err != nil {
		ret.err = errors.Wrap(err, "GetMyBounty WaitMined error!!!")
		res <- ret
		return
	}
	if receipt.Status != 1 {
		ret.err = errors.New("GetMyBounty receipt.Status not equal 1!!!")
		res <- ret
		return
	}
	fmt.Println("GetMyBounty success: ", receipt)

	// // --------------Resolve it later--------------.
	// txOpts, err = bind.NewKeyedTransactorWithChainID(t.key, t.chainID)
	// if err != nil {
	// 	ret.err = errors.Wrap(err, "bind.NewKeyedTransactorWithChainID error!!!")
	// 	res <- ret
	// 	return
	// }
	// pendingNonce, err = t.cli.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(t.key.PublicKey))
	// if err != nil {
	// 	ret.err = errors.Wrap(err, "PendingNonceAt error!!!")
	// 	res <- ret
	// 	return
	// }
	// txOpts.Nonce = big.NewInt(0).SetUint64(pendingNonce)
	// gasPrice, err = t.cli.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	ret.err = errors.Wrap(err, "SuggestGasPrice error!!!")
	// 	res <- ret
	// 	return
	// }
	// txOpts.GasPrice = gasPrice
	// t.randao.RefundBounty(txOpts, campainId)
	// if err != nil {
	// 	ret.err = errors.Wrap(err, "RefundBounty error!!!")
	// 	res <- ret
	// 	return
	// }
	// fmt.Println("tx hash", tx.Hash())
	// receipt, err = bind.WaitMined(context.Background(), t.cli, tx)
	// if err != nil {
	// 	ret.err = errors.Wrap(err, "RefundBounty WaitMined error!!!")
	// 	res <- ret
	// 	return
	// }
	// if receipt.Status != 1 {
	// 	ret.err = errors.New("RefundBounty receipt.Status not equal 1!!!")
	// 	res <- ret
	// 	return
	// }
	// fmt.Println("RefundBounty success: ", receipt)

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
