package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
)

type ProofOfWork struct {
	block *Block

	target *big.Int
}

const Bits  = 16

func NewProofOfWork(block *Block) *ProofOfWork {
	pow := ProofOfWork{
		block: block,
	}

	// 这里是固定的难度值
	/*targetStr := "0001000000000000000000000000000000000000000000000000000000000000"
	var bigIntTmp big.Int
	bigIntTmp.SetString(targetStr, 16)
	pow.target = &bigIntTmp*/

	// 这里是推导的难度值,推导前导为3个难度值
	// 	0001000000000000000000000000000000000000000000000000000000000000
	// 初始化
	//   0000000000000000000000000000000000000000000000000000000000000001
	// 向左移动，256位
	// 1 0000000000000000000000000000000000000000000000000000000000000000
	// 向右移动，四次，一个16进制位代表4个2进制
	// 向右移动16位

	bigIntTmp := big.NewInt(1)
	//bigIntTmp.Lsh(bigIntTmp,256)
	//bigIntTmp.Rsh(bigIntTmp,16)
	bigIntTmp.Lsh(bigIntTmp,256-Bits)

	pow.target = bigIntTmp

	return &pow
}

//这是pow的运算方法，为了获取挖矿的随机数，同时返回区块的哈 希值
func (pow *ProofOfWork) Run() ([]byte,uint64) {
	//1. 获取block数据
	//2. 拼接nonce
	//3. sha256
	//4. 与难度值比较
		//哈希值大于难度值，nonce++
		//哈希 值小于难度值，挖矿成功，退出
	var nonce uint64
	var hash [32]byte

	for {
		fmt.Printf("%x\r",hash)

		hash = sha256.Sum256(pow.PrepareData(nonce))

		// 将hash(数组类型)转换成big.Int类型
		var bigIntTmp big.Int
		bigIntTmp.SetBytes(hash[:])

		//	-1 if x < y
		//	0 if x == y
		//	1 if x > y
		//	func (x *Int) Cmp(y *Int) (r int)
		if bigIntTmp.Cmp(pow.target) == -1 {
			fmt.Printf("Successful mining! nonce: %d,hash: %x\n",nonce,hash)
			break
		} else {
			nonce++
		}
	}

	return hash[:],nonce
}

func (pow *ProofOfWork) PrepareData(nonce uint64) []byte {
	block := pow.block

	tmp := [][]byte{
		uintToByte(block.Version),
		block.PrevBlockHash,
		block.MerkleRoot,
		uintToByte(block.TimeStamp),
		uintToByte(block.Difficulity),
		uintToByte(nonce),
	}

	//更正： 比特币做哈希值，并不是对整个区块做哈希值，而是对区块头做哈希值

	data := bytes.Join(tmp,[]byte{})
	return data
}

func (pow *ProofOfWork) IsValid() bool {
	//在校验的时候，block的数据是完整的，我们要做的是校验一下Hash,block数据，和Nonce是否满足难度值要求
	//1. 获取block数据
	//2. 拼接nonce
	//3. sha256
	//4. 与难度值比较

	data := pow.PrepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)

	var tmp big.Int
	tmp.SetBytes(hash[:])

	return tmp.Cmp(pow.target) == -1
}
