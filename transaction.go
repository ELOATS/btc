package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"github.com/base58"
	"log"
	"math/big"
	"strings"
)

type TXInput struct {
	TXID  []byte //交易ID
	Index int64  //output的索引
	//Address string //解锁脚本，先使用地址模拟

	Signature []byte //交易签名
	PubKey    []byte //公钥本身
}

type TXOutput struct {
	Value float64 //转账金额
	//Address string //锁定脚本

	PubKeyHash []byte //公钥的哈希
}

//给定转账地址，得到这个地址的公钥哈希，完成对output的锁定
func (output *TXOutput) Lock(address string) {
	//25byte
	decodeInfo,_ := base58.Decode(address)

	pubKeyHash := decodeInfo[1:len(decodeInfo)-4]

	output.PubKeyHash = pubKeyHash
}

func NewTXOutput(value float64,address string) TXOutput {
	output := TXOutput{Value:value}
	output.Lock(address)

	return output
}

type Transaction struct {
	TXid      []byte     //交易id
	TXInputs  []TXInput  //所有的inputs
	TXOutputs []TXOutput //所有的outputs
}

func (tx *Transaction) SetTXID() {
	var buffer bytes.Buffer

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	hash := sha256.Sum256(buffer.Bytes())
	tx.TXid = hash[:]
}

const reward  = 12.5

// 实现挖矿交易，特点：只有输出，没有有效的输入(不需要引用id，不需要索引，不需要签名)
// 把挖矿的人传递进来，因为有奖励
func NewCoinbaseTx(miner string, data string) *Transaction {

	inputs := []TXInput{{nil, -1, nil,[]byte(data)}}
	//outputs := []TXOutput{{12.5, miner}}

	output := NewTXOutput(reward,miner)
	outputs := []TXOutput{output}

	tx := Transaction{nil, inputs, outputs}
	tx.SetTXID()

	return &tx
}

func (tx *Transaction) IsCoinbase() bool {
	//特点：只有一个input、引用的id是nil、引用的索引是-1
	inputs := tx.TXInputs

	if len(inputs) == 1 && inputs[0].TXID == nil && inputs[0].Index == -1 {
		return true
	}
	return false
}

/*
实现普通交易
内部逻辑：
1. 遍历账本，找到属于付款人的合适的金额，把这个outputs找到
2. 如果找到钱不足以转账，创建交易失败
3. 将outputs转成inputs
4. 创建输出，创建一个属于收款人的output
5. 如果有找零，创建属于付款人的output
6. 设置交易id
7. 返回交易结构
*/

func NewTransaction(from, to string, amount float64, bc *BlockChain) *Transaction {

	//1. 打开钱包
	ws := NewWallets()
	//获取秘钥对
	wallet := ws.WalletsMap[from]
	if wallet == nil {
		fmt.Printf("%s 的私钥不存在，交易创建失败！\n",from)
		return nil
	}
	//2. 获取公钥，私钥
	publicKey := wallet.PublicKey
	privateKey := wallet.PrivateKey

	pubKeyHash := HashPubKey(wallet.PublicKey)

	utxoes := make(map[string][]int64) //标示能用的utxo
	var resValue float64               //这些utxo存储的金额

	//1. 遍历账本，找到属于付款人的合适的金额，把这个outputs找到
	utxoes, resValue = bc.FindNeedUtxoes(pubKeyHash, amount)

	//2. 如果找到钱不足以转账，创建交易失败
	if resValue < amount {
		fmt.Println("余额不足，交易失败！")
		return nil
	}

	var inputs []TXInput
	var outputs []TXOutput

	//3. 将outputs转成inputs
	for txid, indexes := range utxoes {
		for _, i /*0,1*/ := range indexes {
			input := TXInput{[]byte(txid), i, nil,publicKey}
			inputs = append(inputs, input)
		}
	}

	//4. 创建输出，创建一个属于收款人的output
	//output := TXOutput{amount, to}
	output := NewTXOutput(amount,to)
	outputs = append(outputs, output)

	//5. 如果有找零，创建属于付款人的output
	if resValue > amount {
		//output2 := TXOutput{resValue - amount, from}
		output2 := NewTXOutput(resValue-amount,from)
		outputs = append(outputs, output2)
	}

	//创建交易
	tx := Transaction{nil, inputs, outputs}

	//6. 设置交易id
	tx.SetTXID()

	bc.SignTransaction(&tx,privateKey)


	//7. 返回交易结构
	return &tx
}

//第一个参数是私钥
//第二个参数是这个交易的input所引用的所有的交易
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey,prevTXs map[string]Transaction) {
	fmt.Println("对交易进行签名。。。")

	//校验的时候，如果是挖矿交易，直接返回true
	if tx.IsCoinbase() {
		return
	}

	//1. 拷贝一份交易txCopy
	//   做相应裁剪：把每一个input的Sign和pubKey设置为nil
	//   output不做改变
	txCopy := tx.TrimmedCopy()

	//2. 遍历txCopy.inputs,
	// 把这个input所引用的output的哈希拿过来，赋值给pubKey
	for i,input := range txCopy.TXInputs {
		//找到引用的交易
		preTX := prevTXs[string(input.TXID)]
		output := preTX.TXOutputs[input.Index]

		// for循环迭代出来的数据是一个副本，对这个input进行修改，不会影响到原始数据，
		// 所以我们这里需要使用下标方式修改

		//input.PubKey = output.PubKeyHash
		txCopy.TXInputs[i].PubKey = output.PubKeyHash

		//签名要对数据的hash进行签名
		//我们的数据都在交易中，我们要求交易的哈希
		//Transaction的SetTXID函数就是对交易的哈希
		//所以我们可以使用交易id作为我们的签名的内容

		//3. 生成要签名的数据（哈希）
		txCopy.SetTXID()
		signData := txCopy.TXid

		// 清理
		//input.PubKey = nil
		txCopy.TXInputs[i].PubKey = nil

		fmt.Printf("要签名的数据,signData : %x\n",signData)

		//4. 对数据进行签名r,s
		r,s,err := ecdsa.Sign(rand.Reader,privKey,signData)

		if err != nil {
			fmt.Printf("交易签名失败,err : %v\n",err)
		}

		//5. 拼接r,s为字节流，赋值给原始的交易的Signature字段
		signature := append(r.Bytes(),s.Bytes()...)

		tx.TXInputs[i].Signature = signature

	}
}

//做相应裁剪：把每一个input的Sign和pubKey设置为nil
//output不做改变
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs [] TXInput
	var outputs []TXOutput

	for _,input := range tx.TXInputs {
		input2 := TXInput{input.TXID,input.Index,nil,nil}
		inputs = append(inputs,input2)
	}

	outputs = tx.TXOutputs

	tx2 := Transaction{tx.TXid,inputs,outputs}

	return tx2
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	fmt.Println("对交易进行校验。。。")

	//1. 拷贝裁剪的副本
	txCopy := tx.TrimmedCopy()

	//2. 遍历原始交易（注意：不是txCopy）
	for i,input := range tx.TXInputs {
		//3. 遍历原始交易的input所引用的前交易prevTX
		prevTX := prevTXs[string(input.TXID)]
		output := prevTX.TXOutputs[input.Index]

		//4. 找到output的公钥哈希，赋值给txCopy对应的input
		txCopy.TXInputs[i].PubKey = output.PubKeyHash

		//5. 还原签名的数据
		txCopy.SetTXID()

		// 清空
		tx.TXInputs[i].PubKey = nil

		verifyData := txCopy.TXid
		fmt.Printf("verifyData : %x\n",verifyData)

		//6. 校验
		//还原签名为r,s
		signature := input.Signature

		r := big.Int{}
		s := big.Int{}

		rData := signature[:len(signature)/2]
		sData := signature[len(signature)/2:]

		r.SetBytes(rData)
		s.SetBytes(sData)


		//还原公钥为curve,x,y
		x := big.Int{}
		y := big.Int{}

		// 公钥字节流
		pubKeyBytes := input.PubKey

		xData := pubKeyBytes[:len(pubKeyBytes)/2]
		yData := pubKeyBytes[len(pubKeyBytes)/2:]

		x.SetBytes(xData)
		y.SetBytes(yData)

		curve := elliptic.P256()

		publicKey := ecdsa.PublicKey{curve,&x,&y}

		//数据，签名，公钥准备完毕，开始校验
		if !ecdsa.Verify(&publicKey,verifyData,&r,&s) {
			return false
		}

	}

	return true
}

func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines,fmt.Sprintf("--- Transaction %x\n",tx.TXid))

	for i,input := range tx.TXInputs {
		lines = append(lines,fmt.Sprintf("	 Input: %d",i))
		lines = append(lines,fmt.Sprintf("		TXID:		%x",input.TXID))
		lines = append(lines,fmt.Sprintf("		Out:		%d",input.Index))
		lines = append(lines,fmt.Sprintf("		Signature:	%x",input.Signature))
		lines = append(lines,fmt.Sprintf("		PubKey:		%x",input.PubKey))
	}

	for i,output := range tx.TXOutputs {
		lines = append(lines,fmt.Sprintf("	 Output: %d",i))
		lines = append(lines,fmt.Sprintf("		Value: 		%f",output.Value))
		lines = append(lines,fmt.Sprintf("		Script:		%x",output.PubKeyHash))
	}

	return strings.Join(lines,"\n")
}