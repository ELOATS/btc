package main

import (
	"bytes"
	"fmt"
	"time"
)

func (cli *CLI) CreateBlockChain(addr string) {
	if !IsValidAddress(addr) {
		fmt.Printf("%s 是无效地址!\n",addr)
		return
	}

	bc := CreateBlockChain(addr)
	if bc != nil {
		defer bc.db.Close()
	}

	fmt.Println("Creating blockchain is successful.")
}

func (cli *CLI) GetBalance(addr string) {

	if !IsValidAddress(addr) {
		fmt.Printf("%s 是无效地址!\n",addr)
		return
	}

	bc := NewBlockChain()
	if bc == nil {
		return
	}
	defer bc.db.Close()

	bc.GetBalance(addr)
}

func (cli *CLI) PrintChain() {
	bc := NewBlockChain()
	if bc == nil {
		return
	}
	defer bc.db.Close()

	it := bc.NewIterator()

	for {
		block := it.Next()

		fmt.Printf("+++++++++++++++++++++++++++++++++++++++++NEW BLOCK++++++++++++++++++++++++++++++++++++++++\n")
		fmt.Printf("Version: %v\n", block.Version)
		fmt.Printf("PrevBlockHash: %x\n", block.PrevBlockHash)
		fmt.Printf("MerkleRoot: %x\n", block.MerkleRoot)
		timeFormat := time.Unix(int64(block.TimeStamp), 0).Format("2006-01-02 15:04:05")
		fmt.Printf("TimeStamp: %s\n", timeFormat)
		fmt.Printf("Difficulity: %v\n", block.Difficulity)
		fmt.Printf("Nonce: %v\n", block.Nonce)
		fmt.Printf("Data: %s\n", block.Transactions[0].TXInputs[0].PubKey)
		fmt.Printf("Hash: %x\n", block.Hash)

		pow := NewProofOfWork(block)
		fmt.Printf("IsValid: %v\n", pow.IsValid())

		if bytes.Equal(block.PrevBlockHash, []byte{}) {
			fmt.Println("Traversing blockchain is over!")
			break
		}
	}
}

func (cli *CLI) Send(from,to string,amount float64,miner,data string) {

	if !IsValidAddress(from) {
		fmt.Printf("from : %s 是无效地址!\n",from)
		return
	}

	if !IsValidAddress(to) {
		fmt.Printf("to : %s 是无效地址!\n",to)
		return
	}

	if !IsValidAddress(miner) {
		fmt.Printf("miner : %s 是无效地址!\n",miner)
		return
	}

	bc := NewBlockChain()
	if bc == nil {
		return
	}
	defer bc.db.Close()

	//1. 创建挖矿交易
	coinbase := NewCoinbaseTx(miner,data)

	//创建交易的集合
	txes := []*Transaction{coinbase}

	//2. 创建普通交易
	tx := NewTransaction(from,to,amount,bc)

	if tx != nil {
		txes = append(txes,tx)
	} else {
		fmt.Println("发现无效交易，过滤。")
	}

	//3. 添加到区块
	bc.AddBlock(txes)

	fmt.Println("Mining is successful!")
}

func (cli *CLI) CreateWallet() {

	ws := NewWallets()
	address := ws.CreateWallet()

	fmt.Println("新的钱包地址为: ",address)
}

func (cli *CLI) ListAddresses() {
	ws := NewWallets()

	addresses := ws.ListAddress()

	for _,address := range addresses {
		fmt.Printf("  %v\n",address)
	}
}

func (cli *CLI) PrintTx() {

	bc := NewBlockChain()
	if bc == nil {
		return
	}

	defer bc.db.Close()

	it := bc.NewIterator()

	for {
		block := it.Next()

		fmt.Println("+++++++++++++++++++++++++++++++++++++++NEW BLOCK+++++++++++++++++++++++++++++++++++++++")
		for _,tx := range block.Transactions {
			fmt.Printf("tx : %v\n",tx)
		}

		if len(block.PrevBlockHash) == 0{
			break
		}
	}
}