package main

import (
	"fmt"
	"os"
	"strconv"
)

const Usage = `
Usage:
	./blockchain createBlockChain ADDRESS
	./blockchain printChain
	./blockchain getBalance ADDRESS 
	./blockchain send FROM TO AMOUNT MINER DATA
	./blockchain createWallet 
	./blockchain listAddresses
	./blockchain printTx
`

type CLI struct {
	//bc *BlockChain //
}

func (cli *CLI) Run() {
	cmds := os.Args

	if len(cmds) < 2 {
		fmt.Printf(Usage)
		os.Exit(1)
	}

	switch cmds[1] {
	case "createBlockChain":
		if len(cmds) != 3 {
			fmt.Printf(Usage)
			os.Exit(1)
		}
		addr := cmds[2]
		cli.CreateBlockChain(addr)
	case "printChain":
		cli.PrintChain()
	case "getBalance":
		cli.GetBalance(cmds[2])
	case "send":
		if len(cmds) != 7 {
			fmt.Println("Please check it!")
			fmt.Printf(Usage)
			os.Exit(1)
		}

		from := cmds[2]
		to := cmds[3]
		amount,_ := strconv.ParseFloat(cmds[4],64)
		miner := cmds[5]
		data := cmds[6]

		cli.Send(from,to,amount,miner,data)
	case "createWallet":
		cli.CreateWallet()
	case "listAddresses":
		cli.ListAddresses()
	case "printTx":
		cli.PrintTx()
	default:
		fmt.Println("Please check it.")
		fmt.Printf(Usage)
	}
}
