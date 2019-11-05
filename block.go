package main

import (
	//"bytes"
	//"crypto/sha256"
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"time"
)

const genesisInfo = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"

type Block struct {
	Version       uint64 //区块版本号
	PrevBlockHash []byte //前区块哈希
	MerkleRoot    []byte //先填写为空，v4的时候使用
	TimeStamp     uint64 //从1970.1.1 至今的秒数
	Difficulity   uint64 //挖矿的难度值，v2的时候使用
	Nonce         uint64 //随机数，挖矿找的就是它
	//Data          []byte //数据，目前使用字节 流，v4开始使用交易代替
	Transactions []*Transaction
	Hash          []byte //当前区块哈希，区块中本不存在的字段，为了方便我们添加进来
}

//模拟梅克尔根
func (block *Block) HashTransactions() {
	//我们的交易的id就是交易的哈希值，所以我们可以将交易id拼接起来，整体做 一次哈希运算，作为MerkleRoot
	var hashes []byte

	for _,tx := range block.Transactions {
		txid := tx.TXid
		hashes = append(hashes,txid...)
	}

	hash := sha256.Sum256(hashes)
	block.MerkleRoot = hash[:]
}

func NewBlock(txs []*Transaction, prevBlockHash []byte) *Block {
	block := Block{
		Version:       00,
		PrevBlockHash: prevBlockHash,
		MerkleRoot:    []byte{},
		TimeStamp:     uint64(time.Now().Unix()),
		Difficulity:   Bits, //v2再调整
		Nonce:         10, //同Difficulity
		//Data:          []byte(data),
		Transactions:txs,
		Hash:          []byte{}, //先填充为空
	}
	//block.SetHash()

	block.HashTransactions()

	pow := NewProofOfWork(&block)
	hash,nonce := pow.Run()
	block.Hash = hash
	block.Nonce = nonce

	return &block
}

// 序列化，将区块转换成字节流
func (block *Block) Serialize() []byte {
	var buffer bytes.Buffer

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(block)
	if err != nil {
		log.Panic(err)
	}

	return buffer.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}

/*func (block *Block) SetHash() {
	var data []byte
	//uintToByte将数字转成[]byte{},在utils.go实现
	data = append(data,uintToByte(block.Version)...)
	data = append(data, block.PrevBlockHash...)
	data = append(data,block.MerkleRoot...)
	data = append(data,uintToByte(block.TimeStamp)...)
	data = append(data,uintToByte(block.Difficulity)...)
	data = append(data,uintToByte(block.Nonce)...)
	data = append(data, block.Data...)

	//使用bytes.Join改写函数
	tmp := [][]byte{
		uintToByte(block.Version),
		block.PrevBlockHash,
		block.MerkleRoot,
		uintToByte(block.TimeStamp),
		uintToByte(block.Difficulity),
		uintToByte(block.Nonce),
		block.Data,
	}

	data := bytes.Join(tmp,[]byte{})

	//hash: [32]byte
	hash := sha256.Sum256(data)
	block.Hash = hash[:]
}*/
