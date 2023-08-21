package main

import (
	"context"
	"encoding/json"
	contract "findora/randao/contract"
	model "findora/randao/model"
	utils "findora/randao/utils"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

var CONF_FILE = "config_debug0.json"

func main() {
	fmt.Println("campaign main")

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println(dir)

	conf_str, err := os.ReadFile(CONF_FILE)
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

	randao, err := contract.NewRandao(common.HexToAddress(conf.Chain.Randao), cli)
	if err != nil {
		panic(fmt.Sprintf("contract.NewRandao error: %s", err.Error()))
	}
	fmt.Println("Randao address: ", conf.Chain.Randao)

	randaoAbi, err := abi.JSON(strings.NewReader(contract.RandaoABI))
	if err != nil {
		panic(fmt.Sprintf("abi.JSON error: %s", err.Error()))
	}

	chainID, err := cli.ChainID(context.Background())
	if err != nil {
		panic(fmt.Sprintf("cli.ChainID error: %s", err.Error()))
	}
	privateKeyECDSA, err := crypto.HexToECDSA(conf.Chain.Campaigner)
	if err != nil {
		panic(fmt.Sprintf("crypto.HexToECDSA error: %s", err.Error()))
	}
	fmt.Println("privateKeyECDSA address: ", conf.Chain.Campaigner)

	block_num, err := cli.BlockNumber(context.Background())
	if err != nil {
		panic(fmt.Sprintf("cli.BlockNumber error: %s", err.Error()))
	}
	var bnum *big.Int = big.NewInt(0).SetUint64(block_num + 20)
	var deposit *big.Int = big.NewInt(1000_000_000_000_000_000)
	var commit_balkline uint16 = 16
	var commit_deadline uint16 = 8
	var maxFee *big.Int = big.NewInt(10_000_000_000_000_000)
	data, err := randaoAbi.Pack("newCampaign", bnum, deposit, commit_balkline, commit_deadline, maxFee)
	if err != nil {
		panic(fmt.Sprintf("randaoAbi.Pack error: %s", err.Error()))
	}

	to := common.HexToAddress(conf.Chain.Randao)
	estimateGasCall := ethereum.CallMsg{
		To:    &to,
		Data:  data,
		Value: deposit,
	}

	gasLimit, err := cli.EstimateGas(context.Background(), estimateGasCall)
	if err != nil {
		log.Fatal(errors.Wrap(err, "EstimateGas error:"))
	}

	gasPrice, err := cli.SuggestGasPrice(context.Background())
	if err != nil {
		panic(fmt.Sprintf("cli.SuggestGasPrice error: %s", err.Error()))
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(100))
	gasLimit = gasLimit * 100

	fmt.Println("Estimate GasLimit Gas:", gasLimit)
	fmt.Println("Estimate GasPrice Gas:", gasPrice)

	// opts.GasPrice = gasPrice
	// opts.GasLimit = gasLimit
	opts, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, chainID)
	if err != nil {
		panic(fmt.Sprintf("bind.NewKeyedTransactorWithChainID error: %s", err.Error()))
	}
	opts.Value = deposit
	tx, err := randao.NewCampaign(opts, bnum, deposit, commit_balkline, commit_deadline, maxFee)
	if err != nil {
		panic(fmt.Sprintf("randao.NewCampaign error: %s", err.Error()))
	}
	fmt.Println("tx hash", tx.Hash())

	receipt, err := bind.WaitMined(context.Background(), cli, tx)
	if err != nil {
		panic(fmt.Sprintf("cli.TransactionReceipt error: %s", err.Error()))
	}
	if receipt.Status != 1 {
		panic("receipt.Status not equal 1")
	}
	fmt.Println("receipt: ", receipt)

	log := receipt.Logs[0]
	ret1, err := randao.ParseLogCampaignAdded(*log)
	if err != nil {
		panic(fmt.Sprintf("ParseLogCampaignAdded error: %s", err.Error()))
	}
	fmt.Println("event LogCampaignAdded: ", ret1)

	fmt.Println("campaignId: ", ret1.CampaignID)

	utils.WaitBlocks(cli, bnum.Uint64())

	opts, err = bind.NewKeyedTransactorWithChainID(privateKeyECDSA, chainID)
	if err != nil {
		panic(fmt.Sprintf("bind.NewKeyedTransactorWithChainID error: %s", err.Error()))
	}
	tx, err = randao.GetRandom(
		opts,
		ret1.CampaignID)
	if err != nil {
		panic(fmt.Sprintf("randao.GetRandom error: %s", err.Error()))
	}
	fmt.Println("tx hash", tx.Hash())

	receipt, err = bind.WaitMined(context.Background(), cli, tx)
	if err != nil {
		panic(fmt.Sprintf("cli.TransactionReceipt error: %s", err.Error()))
	}
	if receipt.Status != 1 {
		panic("receipt.Status not equal 1")
	}
	fmt.Println("receipt: ", receipt)

	log = receipt.Logs[0]
	ret2, err := randao.ParseLogGetRandom(*log)
	if err != nil {
		panic(fmt.Sprintf("ParseLogGetRandom error: %s", err.Error()))
	}
	fmt.Println("event LogGetRandom: ", ret2)

	fmt.Println("random: ", ret2.Random)
}
