package main

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/base58"
	"github.com/boltdb/bolt"
	"log"
	"os"
)

type BlockChain struct {
	db   *bolt.DB
	tail []byte //最后一个区块的哈希
}

const blockChainName = "blockChain.db"
const blockBucketName = "blockBucket"
const lastHashKey = "lastHashKey"

func CreateBlockChain(miner string) *BlockChain {

	if IsFileExist(blockChainName) {
		fmt.Println("区块链已经存在，不需要重复创建!")
		return nil
	}

	//1. 获得数据库的句柄，打开数据库，填写数据
	db, err := bolt.Open(blockChainName, 0600, nil)
	if err != nil {
		log.Panic(err)
	}
	//defer db.Close()

	var tail []byte

	//判断是否有bucket,如果没有，创建bucket
	_ = db.Update(func(tx *bolt.Tx) error {

		b, err := tx.CreateBucket([]byte(blockBucketName))
		if err != nil {
			log.Panic(err)
		}

		//写入创始块
		//创始块中只有一个挖矿交易，只有Coinbase
		coinbase := NewCoinbaseTx(miner, genesisInfo)
		genesisBlock := NewBlock([]*Transaction{coinbase}, []byte{})

		_ = b.Put(genesisBlock.Hash, genesisBlock.Serialize() /*将区块序列化，转成字节流*/)
		//写入lastHashKey这条数据
		_ = b.Put([]byte(lastHashKey), genesisBlock.Hash)

		/*blockInfo := b.Get(genesisBlock.Hash)
		block := Deserialize(blockInfo)
		fmt.Printf("Decoded block data:  %s\n",block)*/

		//更新tail为最后一个区块的哈希
		tail = genesisBlock.Hash

		return nil
	})

	//返回bc实例
	return &BlockChain{db, tail}
}

//返回区块链实例
func NewBlockChain() *BlockChain {

	if !IsFileExist(blockChainName) {
		fmt.Println("区块链不存在，请先创建!")
		return nil
	}

	//1. 获得数据库的句柄，打开数据库，填写数据
	db, err := bolt.Open(blockChainName, 0600, nil)
	if err != nil {
		log.Panic(err)
	}
	//defer db.Close()

	var tail []byte

	//判断是否有bucket,如果没有，创建bucket
	_ = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucketName))

		if b == nil {
			fmt.Println("Bucket is empty,please check it!")
			os.Exit(1)
		}

		tail = b.Get([]byte(lastHashKey))

		return nil
	})

	//返回bc实例
	return &BlockChain{db, tail}
}

func (bc *BlockChain) AddBlock(txs []*Transaction) {
	//矿工得到交易时，第一时间对交易进行验证
	//矿工如果不验证，即使挖矿成功，广播区块后，其他的验证矿工，仍然会检验每一笔交易

	validTXs := []*Transaction{}

	for _,tx := range txs {
		if bc.VerifyTransaction(tx) {
			fmt.Printf("--- 该交易有效: %x\n",tx.TXid)
			validTXs = append(validTXs,tx)
		} else {
			fmt.Printf("发现无效的交易: %x\n",tx.TXid)
		}
	}



	_ = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blockBucket"))

		if b == nil {
			fmt.Println("Bucket does not exist,please check it.")
			os.Exit(1)
		}

		block := NewBlock(txs, bc.tail)
		_ = b.Put(block.Hash, block.Serialize())
		_ = b.Put([]byte("lastHashKey"), block.Hash)

		bc.tail = block.Hash

		return nil
	})
}

// 定义一个区块链年的迭代器，包括db,current
type BlockChainIterator struct {
	db      *bolt.DB
	current []byte //当前所指向区块的哈希值
}

// 当前所指向区块的哈希值
func (bc *BlockChain) NewIterator() *BlockChainIterator {
	return &BlockChainIterator{bc.db, bc.tail}
}

func (it *BlockChainIterator) Next() *Block {
	var block Block

	_ = it.db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(blockBucketName))
		if b == nil {
			fmt.Println("Bucket does not exist,please check it.")
			os.Exit(1)
		}

		blockInfo := b.Get(it.current)
		block = *Deserialize(blockInfo)

		it.current = block.PrevBlockHash

		return nil
	})

	return &block
}

//我们想把FindMyUtxoes和FindNeedUTXO进行整合
//1. FindMyUtxoes: 找到所有utxo（只要output就可以了）
//2. FindNeedUTXO: 找到需要的utxo（要output的定位）

//我们可以定义一个结构，同时包含output已经定位信息
type UTXOInfo struct {
	TXID   []byte   //交易id
	Index  int64    //output的索引值
	Output TXOutput //output本身
}

//实现思路：
func (bc *BlockChain) FindMyUtxoes(pubKeyHash []byte) []UTXOInfo {
	//var UTXOes []TXOutput //返回的结构
	var UTXOInfoes []UTXOInfo //新的返回结构

	it := bc.NewIterator()

	//这是标示已经消耗过的utxo的结构，key是交易id，value是这个id里面的outputs索引的数组
	spentUTXOes := make(map[string][]int64)

	//1. 遍历账本
	for {
		block := it.Next()

		//2. 遍历交易
		for _, tx := range block.Transactions {

			// 遍历input
			if tx.IsCoinbase() == false {
				//如果不是coinbase，说明是普通交易，才有必要进行遍历
				for _, input := range tx.TXInputs {

					//判断当前被使用input是否为目标地址所有
					if bytes.Equal(HashPubKey(input.PubKey),pubKeyHash) {

						fmt.Printf("找到了消耗过的output! index: %d\n", input.Index)
						key := string(input.TXID)
						spentUTXOes[key] = append(spentUTXOes[key], input.Index)
					}
				}
			}

			key := string(tx.TXid)
			indexes /*[]int64{0,1}*/ := spentUTXOes[key]

		OUTPUT:
			//3. 遍历output
			for i, output := range tx.TXOutputs {

				if len(indexes) != 0 {
					fmt.Println("当前这笔交易中有被消耗过的output")

					for _, j /*0,1*/ := range indexes {
						if int64(i) == j {
							fmt.Println("i==j, 当前的output已经被消耗过了，跳过不统计!")
							continue OUTPUT
						}
					}
				}

				//4. 找到属于我的所有output
				if bytes.Equal(pubKeyHash,output.PubKeyHash) {
					//fmt.Printf("找到了属于%s 的output,i: %d\n", address, i)
					//UTXOes = append(UTXOes, output)
					utxoinfo := UTXOInfo{tx.TXid, int64(i), output}
					UTXOInfoes = append(UTXOInfoes, utxoinfo)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			fmt.Println("遍历区块链结束!")
			break
		}
	}

	return UTXOInfoes
}

func (bc *BlockChain) GetBalance(address string) {


	// 这个过程，不要打开钱包，因为有可能查看余额的人不是地址本人
	decodeInfo,_ := base58.Decode(address)
	pubKeyHash := decodeInfo[1:len(decodeInfo)-4]

	utxoinfoes := bc.FindMyUtxoes(pubKeyHash)

	var total = 0.0
	//所有的output都在utxoinfoes内部
	//获取余额时，遍历utxoinfoes获取output即可
	for _, utxoinfo := range utxoinfoes {
		total += utxoinfo.Output.Value
	}

	fmt.Printf("%s 的余额为: %f\n", address, total)
}

func (bc *BlockChain) FindNeedUtxoes(pubKeyHash []byte, amount float64) (map[string][]int64, float64) {

	needUtxoes := make(map[string][]int64) //标示能用的utxo
	var resValue float64                   //返回的金额

	//复用FindMyUtxo方法，这个方法已经包含了所有信息
	utxoinfoes := bc.FindMyUtxoes(pubKeyHash)

	for _, utxoinfo := range utxoinfoes {
		key := string(utxoinfo.TXID)

		needUtxoes[key] = append(needUtxoes[key], int64(utxoinfo.Index))
		resValue += utxoinfo.Output.Value

		if resValue >= amount {
			break
		}
	}

	return needUtxoes, resValue
}

func (bc *BlockChain) SignTransaction(tx *Transaction,privateKey *ecdsa.PrivateKey) {
	//遍历账本找到所有应用交易
	prevTXs := make(map[string]Transaction)

	//遍历tx的inputs，通过id去查找所引用的交易
	for _,input := range tx.TXInputs {
		prevTX := bc.FindTransaction(input.TXID)

		if prevTX == nil {
			fmt.Println("没有找到交易",input.TXID)
		} else {
			//把找到的引用交易保存起来
			prevTXs[string(input.TXID)] = *prevTX
		}
	}

	tx.Sign(privateKey,prevTXs)
}

//矿工校验流程：
//1. 找到交易input所引用的所有的交易prevTXs
//2. 对交易进行校验

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {

	//校验的时候，如果是挖矿交易，直接返回true
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)

	//遍历tx的inputs，通过id去查找所引用的交易
	for _,input := range tx.TXInputs {
		prevTX := bc.FindTransaction(input.TXID)

		if prevTX == nil {
			fmt.Println("没有找到交易",input.TXID)
		} else {
			//把找到的引用交易保存起来
			prevTXs[string(input.TXID)] = *prevTX
		}
	}

	return tx.Verify(prevTXs)
}

func (bc *BlockChain) FindTransaction(txid []byte) *Transaction {
	//遍历区块链的交易
	//通过对比id来识别

	it := bc.NewIterator()

	for {
		block := it.Next()

		for _,tx := range block.Transactions {
			//如果找到相同id交易，直接返回交易即可
			if bytes.Equal(tx.TXid,txid) {
				return tx
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return nil
}
