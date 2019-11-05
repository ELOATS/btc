package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
)

//Wallets结构
//把地址和秘钥对对应起来
//map[address1] -> walletKeyPair1
//map[address2] -> walletKeyPair2

type Wallets struct {
	WalletsMap map[string]*WalletKeyPair
}

func NewWallets() *Wallets {
	var ws Wallets

	ws.WalletsMap = make(map[string]*WalletKeyPair)

	//把所有的钱包从本地加载出来
	ws.LoadFromFile()

	//把实例返回
	return &ws
}

const WalletName = "wallet.dat"

//这个Wallets是对外的，WalletKeyPair是对内的
//Wallets调用WalletKeyPai
func (ws *Wallets) CreateWallet() string {
	//调用NewWalletkeyPair
	wallet := NewWalletKeyPair()

	//将返回的walletKeyPair添加到WalletMap中
	address := wallet.GetAddress()

	ws.WalletsMap[address] = wallet

	//保存到本地文件
	res := ws.SaveToFile()
	if !res {
		fmt.Println("保存文件失败")
		return ""
	}

	return address
}

func (ws *Wallets) SaveToFile() bool {
	var buffer bytes.Buffer

	//将接口类型明确注册一下，否则gob编码失败
	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(ws)
	if err != nil {
		fmt.Println("钱包序列化失败!",err)
		return false
	}

	content := buffer.Bytes()

	err = ioutil.WriteFile(WalletName,content,0600)
	if err != nil {
		fmt.Println("钱包创建失败")
		return false
	}

	return true
}

func (ws *Wallets) LoadFromFile() bool {
	//判断文件是否存在
	if !IsFileExist(WalletName) {
		fmt.Println("wallet.dat does not exist.")
		return true
	}

	//read file
	content,err := ioutil.ReadFile(WalletName)
	if err != nil {
		return false
	}

	gob.Register(elliptic.P256())

	decoder := gob.NewDecoder(bytes.NewReader(content))

	var wallets Wallets
	err = decoder.Decode(&wallets)

	if err != nil {
		fmt.Println(err)
		return false
	}

	ws.WalletsMap = wallets.WalletsMap

	return true
}

func (ws *Wallets) ListAddress() []string {
	//遍历ws.WalletsMap结构返回key即可
	var addresses []string

	for address,_ := range ws.WalletsMap {
		addresses = append(addresses,address)
	}

	return addresses
}