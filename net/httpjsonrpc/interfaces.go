package httpjsonrpc

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"UNetwork/account"
	. "UNetwork/common"
	"UNetwork/common/config"
	"UNetwork/common/log"
	"UNetwork/core/forum"
	"UNetwork/core/ledger"
	"UNetwork/core/signature"
	tx "UNetwork/core/transaction"
	. "UNetwork/errors"
	"UNetwork/sdk"


	"github.com/mitchellh/go-homedir"
)

const (
	RANDBYTELEN = 4
)

func TransArryByteToHexString(ptx *tx.Transaction) *Transactions {

	trans := new(Transactions)
	trans.TxType = ptx.TxType
	trans.PayloadVersion = ptx.PayloadVersion
	trans.Payload = TransPayloadToHex(ptx.Payload)

	n := 0
	trans.Attributes = make([]TxAttributeInfo, len(ptx.Attributes))
	for _, v := range ptx.Attributes {
		trans.Attributes[n].Usage = v.Usage
		trans.Attributes[n].Data = BytesToHexString(v.Data)
		n++
	}

	n = 0
	trans.UTXOInputs = make([]UTXOTxInputInfo, len(ptx.UTXOInputs))
	for _, v := range ptx.UTXOInputs {
		trans.UTXOInputs[n].ReferTxID = BytesToHexString(v.ReferTxID.ToArrayReverse())
		trans.UTXOInputs[n].ReferTxOutputIndex = v.ReferTxOutputIndex
		n++
	}

	n = 0
	trans.BalanceInputs = make([]BalanceTxInputInfo, len(ptx.BalanceInputs))
	for _, v := range ptx.BalanceInputs {
		trans.BalanceInputs[n].AssetID = BytesToHexString(v.AssetID.ToArrayReverse())
		trans.BalanceInputs[n].Value = v.Value.String()
		trans.BalanceInputs[n].ProgramHash = BytesToHexString(v.ProgramHash.ToArrayReverse())
		n++
	}

	n = 0
	trans.Outputs = make([]TxoutputInfo, len(ptx.Outputs))
	for _, v := range ptx.Outputs {
		trans.Outputs[n].AssetID = BytesToHexString(v.AssetID.ToArrayReverse())
		trans.Outputs[n].Value = v.Value.String()
		address, _ := v.ProgramHash.ToAddress()
		trans.Outputs[n].Address = address
		n++
	}

	n = 0
	trans.Programs = make([]ProgramInfo, len(ptx.Programs))
	for _, v := range ptx.Programs {
		trans.Programs[n].Code = BytesToHexString(v.Code)
		trans.Programs[n].Parameter = BytesToHexString(v.Parameter)
		n++
	}

	n = 0
	trans.AssetOutputs = make([]TxoutputMap, len(ptx.AssetOutputs))
	for k, v := range ptx.AssetOutputs {
		trans.AssetOutputs[n].Key = k
		trans.AssetOutputs[n].Txout = make([]TxoutputInfo, len(v))
		for m := 0; m < len(v); m++ {
			trans.AssetOutputs[n].Txout[m].AssetID = BytesToHexString(v[m].AssetID.ToArrayReverse())
			trans.AssetOutputs[n].Txout[m].Value = v[m].Value.String()
			address, _ := v[m].ProgramHash.ToAddress()
			trans.AssetOutputs[n].Txout[m].Address = address
		}
		n += 1
	}

	n = 0
	trans.AssetInputAmount = make([]AmountMap, len(ptx.AssetInputAmount))
	for k, v := range ptx.AssetInputAmount {
		trans.AssetInputAmount[n].Key = k
		trans.AssetInputAmount[n].Value = v
		n += 1
	}

	n = 0
	trans.AssetOutputAmount = make([]AmountMap, len(ptx.AssetOutputAmount))
	for k, v := range ptx.AssetOutputAmount {
		trans.AssetInputAmount[n].Key = k
		trans.AssetInputAmount[n].Value = v
		n += 1
	}

	mHash := ptx.Hash()
	trans.Hash = BytesToHexString(mHash.ToArrayReverse())

	return trans
}
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}
func getBestBlockHash(params []interface{}) map[string]interface{} {
	hash := ledger.DefaultLedger.Blockchain.CurrentBlockHash()
	return UnetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

// Input JSON string examples for getblock method as following:
//   {"jsonrpc": "2.0", "method": "getblock", "params": [1], "id": 0}
//   {"jsonrpc": "2.0", "method": "getblock", "params": ["aabbcc.."], "id": 0}
func getBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	var err error
	var hash Uint256
	switch (params[0]).(type) {
	// block height
	case float64:
		index := uint32(params[0].(float64))
		hash, err = ledger.DefaultLedger.Store.GetBlockHash(index)
		if err != nil {
			return UnetworkRpcUnknownBlock
		}
	// block hash
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return UnetworkRpcInvalidParameter
		}
		if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
			return UnetworkRpcInvalidTransaction
		}
	default:
		return UnetworkRpcInvalidParameter
	}

	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return UnetworkRpcUnknownBlock
	}

	blockHead := &BlockHead{
		Version:          block.Blockdata.Version,
		PrevBlockHash:    BytesToHexString(block.Blockdata.PrevBlockHash.ToArrayReverse()),
		TransactionsRoot: BytesToHexString(block.Blockdata.TransactionsRoot.ToArrayReverse()),
		Timestamp:        block.Blockdata.Timestamp,
		Height:           block.Blockdata.Height,
		ConsensusData:    block.Blockdata.ConsensusData,
		NextBookKeeper:   BytesToHexString(block.Blockdata.NextBookKeeper.ToArrayReverse()),
		Program: ProgramInfo{
			Code:      BytesToHexString(block.Blockdata.Program.Code),
			Parameter: BytesToHexString(block.Blockdata.Program.Parameter),
		},
		Hash: BytesToHexString(hash.ToArrayReverse()),
	}

	trans := make([]*Transactions, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		trans[i] = TransArryByteToHexString(block.Transactions[i])
	}

	b := BlockInfo{
		Hash:         BytesToHexString(hash.ToArrayReverse()),
		BlockData:    blockHead,
		Transactions: trans,
	}
	return UnetworkRpc(b)
}

func getBlockCount(params []interface{}) map[string]interface{} {
	return UnetworkRpc(ledger.DefaultLedger.Blockchain.BlockHeight + 1)
}

// A JSON example for getblockhash method as following:
//   {"jsonrpc": "2.0", "method": "getblockhash", "params": [1], "id": 0}
func getBlockHash(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	switch params[0].(type) {
	case float64:
		height := uint32(params[0].(float64))
		hash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
		if err != nil {
			return UnetworkRpcUnknownBlock
		}
		return UnetworkRpc(BytesToHexString(hash.ToArrayReverse()))
	default:
		return UnetworkRpcInvalidParameter
	}
}

func getConnectionCount(params []interface{}) map[string]interface{} {
	return UnetworkRpc(node.GetConnectionCnt())
}

func getRawMemPool(params []interface{}) map[string]interface{} {
	txs := []*Transactions{}
	txpool := node.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArryByteToHexString(t))
	}
	if len(txs) == 0 {
		return UnetworkRpcNil
	}
	return UnetworkRpc(txs)
}

// A JSON example for getrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "getrawtransaction", "params": ["transactioin hash in hex"], "id": 0}
func getRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return UnetworkRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return UnetworkRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return UnetworkRpcUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		return UnetworkRpc(tran)
	default:
		return UnetworkRpcInvalidParameter
	}
}

// A JSON example for sendrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "sendrawtransaction", "params": ["raw transactioin in hex"], "id": 0}
func sendRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytes(str)
		if err != nil {
			return UnetworkRpcInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return UnetworkRpcInvalidTransaction
		}

		//if txn.TxType != tx.InvokeCode && txn.TxType != tx.DeployCode &&
		//	txn.TxType != tx.TransferAsset && txn.TxType != tx.LockAsset &&
		//	txn.TxType != tx.BookKeeper {
		//	return UnetworkRpc("invalid transaction type")
		//}
		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return UnetworkRpc(errCode.Error())
		}
	default:
		return UnetworkRpcInvalidParameter
	}
	return UnetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

func getTxout(params []interface{}) map[string]interface{} {
	//TODO
	return UnetworkRpcUnsupported
}

// A JSON example for submitblock method as following:
//   {"jsonrpc": "2.0", "method": "submitblock", "params": ["raw block in hex"], "id": 0}
func submitBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, _ := HexStringToBytes(str)
		var block ledger.Block
		if err := block.Deserialize(bytes.NewReader(hex)); err != nil {
			return UnetworkRpcInvalidBlock
		}
		if err := ledger.DefaultLedger.Blockchain.AddBlock(&block); err != nil {
			return UnetworkRpcInvalidBlock
		}
		if err := node.LocalNode().CleanSubmittedTransactions(&block); err != nil {
			return UnetworkRpcInternalError
		}
		if err := node.Xmit(&block); err != nil {
			return UnetworkRpcInternalError
		}
	default:
		return UnetworkRpcInvalidParameter
	}
	return UnetworkRpcSuccess
}

func getNeighbor(params []interface{}) map[string]interface{} {
	addr, _ := node.GetNeighborAddrs()
	return UnetworkRpc(addr)
}

func getNodeState(params []interface{}) map[string]interface{} {
	n := NodeInfo{
		State:    uint(node.GetState()),
		Time:     node.GetTime(),
		Port:     node.GetPort(),
		ID:       node.GetID(),
		Version:  node.Version(),
		Services: node.Services(),
		Relay:    node.GetRelay(),
		Height:   node.GetHeight(),
		TxnCnt:   node.GetTxnCnt(),
		RxTxnCnt: node.GetRxTxnCnt(),
	}
	return UnetworkRpc(n)
}

func startConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Start(); err != nil {
		return UnetworkRpcFailed
	}
	return UnetworkRpcSuccess
}

func stopConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Halt(); err != nil {
		return UnetworkRpcFailed
	}
	return UnetworkRpcSuccess
}

func sendSampleTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	var txType string
	switch params[0].(type) {
	case string:
		txType = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}

	issuer, err := account.NewAccount()
	if err != nil {
		return UnetworkRpc("Failed to create account")
	}
	admin := issuer

	rbuf := make([]byte, RANDBYTELEN)
	rand.Read(rbuf)
	switch string(txType) {
	case "perf":
		num := 1
		if len(params) == 2 {
			switch params[1].(type) {
			case float64:
				num = int(params[1].(float64))
			}
		}
		for i := 0; i < num; i++ {
			regTx := NewRegTx(BytesToHexString(rbuf), i, admin, issuer)
			SignTx(admin, regTx)
			VerifyAndSendTx(regTx)
		}
		return UnetworkRpc(fmt.Sprintf("%d transaction(s) was sent", num))
	default:
		return UnetworkRpc("Invalid transacion type")
	}
}

func setDebugInfo(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcInvalidParameter
	}
	switch params[0].(type) {
	case float64:
		level := params[0].(float64)
		if err := log.Log.SetDebugLevel(int(level)); err != nil {
			return UnetworkRpcInvalidParameter
		}
	default:
		return UnetworkRpcInvalidParameter
	}
	return UnetworkRpcSuccess
}

func getVersion(params []interface{}) map[string]interface{} {
	return UnetworkRpc(config.Version)
}

func uploadDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}

	rbuf := make([]byte, 4)
	rand.Read(rbuf)
	tmpname := hex.EncodeToString(rbuf)

	str := params[0].(string)

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	f, err := os.OpenFile(tmpname, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		return UnetworkRpcIOError
	}
	defer f.Close()
	f.Write(data)

	refpath, err := AddFileIPFS(tmpname, true)
	if err != nil {
		return UnetworkRpcAPIError
	}

	return UnetworkRpc(refpath)

}

func regDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytes(str)
		if err != nil {
			return UnetworkRpcInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return UnetworkRpcInvalidTransaction
		}

		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return UnetworkRpcInternalError
		}
	default:
		return UnetworkRpcInvalidParameter
	}
	return UnetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

func catDataRecord(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		b, err := HexStringToBytesReverse(str)
		if err != nil {
			return UnetworkRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(b))
		if err != nil {
			return UnetworkRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return UnetworkRpcUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)
		//ref := string(record.RecordData[:])
		return UnetworkRpc(info)
	default:
		return UnetworkRpcInvalidParameter
	}
}

func getDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return UnetworkRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return UnetworkRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return UnetworkRpcUnknownTransaction
		}

		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)

		err = GetFileIPFS(info.IPFSPath, info.Filename)
		if err != nil {
			return UnetworkRpcAPIError
		}
		//TODO: shoud return download address
		return UnetworkRpcSuccess
	default:
		return UnetworkRpcInvalidParameter
	}
}

var Wallet account.Client

func getWalletDir() string {
	home, _ := homedir.Dir()
	return home + "/.wallet/"
}

func createWallet(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	var password []byte
	switch params[0].(type) {
	case string:
		password = []byte(params[0].(string))
	default:
		return UnetworkRpcInvalidParameter
	}
	walletDir := getWalletDir()
	if !FileExisted(walletDir) {
		err := os.MkdirAll(walletDir, 0755)
		if err != nil {
			return UnetworkRpcInternalError
		}
	}
	walletPath := walletDir + "wallet.dat"
	if FileExisted(walletPath) {
		return UnetworkRpcWalletAlreadyExists
	}
	_, err := account.Create(walletPath, password)
	if err != nil {
		return UnetworkRpcFailed
	}
	return UnetworkRpcSuccess
}

func openWallet(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	var password []byte
	switch params[0].(type) {
	case string:
		password = []byte(params[0].(string))
	default:
		return UnetworkRpcInvalidParameter
	}
	resp := make(map[string]string)
	walletPath := getWalletDir() + "wallet.dat"
	if !FileExisted(walletPath) {
		resp["success"] = "false"
		resp["message"] = "wallet doesn't exist"
		return UnetworkRpc(resp)
	}
	wallet, err := account.Open(walletPath, password)
	if err != nil {
		resp["success"] = "false"
		resp["message"] = "password wrong"
		return UnetworkRpc(resp)
	}
	Wallet = wallet
	programHash, err := wallet.LoadStoredData("ProgramHash")
	if err != nil {
		resp["success"] = "false"
		resp["message"] = "wallet file broken"
		return UnetworkRpc(resp)
	}
	resp["success"] = "true"
	resp["message"] = BytesToHexString(programHash)
	return UnetworkRpc(resp)
}

func closeWallet(params []interface{}) map[string]interface{} {
	Wallet = nil
	return UnetworkRpcSuccess
}

func recoverWallet(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UnetworkRpcNil
	}
	var privateKey string
	var walletPassword string
	switch params[0].(type) {
	case string:
		privateKey = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		walletPassword = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	walletDir := getWalletDir()
	if !FileExisted(walletDir) {
		err := os.MkdirAll(walletDir, 0755)
		if err != nil {
			return UnetworkRpcInternalError
		}
	}
	walletName := fmt.Sprintf("wallet-%s-recovered.dat", time.Now().Format("2006-01-02-15-04-05"))
	walletPath := walletDir + walletName
	if FileExisted(walletPath) {
		return UnetworkRpcWalletAlreadyExists
	}
	_, err := account.Recover(walletPath, []byte(walletPassword), privateKey)
	if err != nil {
		return UnetworkRpc("wallet recovery failed")
	}

	return UnetworkRpcSuccess
}

func getWalletKey(params []interface{}) map[string]interface{} {
	if Wallet == nil {
		return UnetworkRpc("open wallet first")
	}
	account, _ := Wallet.GetDefaultAccount()
	encodedPublickKey, _ := account.PublicKey.EncodePoint(true)
	resp := make(map[string]string)
	resp["PublicKey"] = BytesToHexString(encodedPublickKey)
	resp["PrivateKey"] = BytesToHexString(account.PrivateKey)
	resp["ProgramHash"] = BytesToHexString(account.ProgramHash.ToArrayReverse())

	return UnetworkRpc(resp)
}

func addAccount(params []interface{}) map[string]interface{} {
	if Wallet == nil {
		return UnetworkRpc("open wallet first")
	}
	account, err := Wallet.CreateAccount()
	if err != nil {
		return UnetworkRpc("create account error:" + err.Error())
	}

	if err := Wallet.CreateContract(account); err != nil {
		return UnetworkRpc("create contract error:" + err.Error())
	}

	address, err := account.ProgramHash.ToAddress()
	if err != nil {
		return UnetworkRpc("generate address error:" + err.Error())
	}

	return UnetworkRpc(address)
}

func deleteAccount(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	var address string
	switch params[0].(type) {
	case string:
		address = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return UnetworkRpc("open wallet first")
	}
	programHash, err := ToScriptHash(address)
	if err != nil {
		return UnetworkRpc("invalid address:" + err.Error())
	}
	if err := Wallet.DeleteAccount(programHash); err != nil {
		return UnetworkRpc("Delete account error:" + err.Error())
	}
	if err := Wallet.DeleteContract(programHash); err != nil {
		return UnetworkRpc("Delete contract error:" + err.Error())
	}
	if err := Wallet.DeleteCoinsData(programHash); err != nil {
		return UnetworkRpc("Delete coins error:" + err.Error())
	}

	return UnetworkRpc(true)
}

func makeRegTxn(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UnetworkRpcNil
	}
	var assetName, assetValue string
	switch params[0].(type) {
	case string:
		assetName = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		assetValue = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return UnetworkRpc("open wallet first")
	}

	regTxn, err := sdk.MakeRegTransaction(Wallet, assetName, assetValue)
	if err != nil {
		return UnetworkRpcInternalError
	}

	if errCode := VerifyAndSendTx(regTxn); errCode != ErrNoError {
		return UnetworkRpcInvalidTransaction
	}
	return UnetworkRpc(true)
}

func makeIssueTxn(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UnetworkRpcNil
	}
	var asset, value, address string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		value = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return UnetworkRpc("open wallet first")
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return UnetworkRpc("invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UnetworkRpc("invalid asset hash")
	}
	issueTxn, err := sdk.MakeIssueTransaction(Wallet, assetID, address, value)
	if err != nil {
		return UnetworkRpcInternalError
	}

	if errCode := VerifyAndSendTx(issueTxn); errCode != ErrNoError {
		return UnetworkRpcInvalidTransaction
	}

	return UnetworkRpc(true)
}

func sendToAddress(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UnetworkRpcNil
	}
	var asset, address, value string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		address = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		value = params[2].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return UnetworkRpc("error : wallet is not opened")
	}

	batchOut := sdk.BatchOut{
		Address: address,
		Value:   value,
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return UnetworkRpc("error: invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UnetworkRpc("error: invalid asset hash")
	}
	txn, err := sdk.MakeTransferTransaction(Wallet, assetID, batchOut)
	if err != nil {
		return UnetworkRpc("error: " + err.Error())
	}

	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UnetworkRpc("error: " + errCode.Error())
	}
	txHash := txn.Hash()
	return UnetworkRpc(BytesToHexString(txHash.ToArrayReverse()))
}

func lockAsset(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UnetworkRpcNil
	}
	var asset, value string
	var height float64
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		value = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case float64:
		height = params[2].(float64)
	default:
		return UnetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return UnetworkRpc("error: invalid wallet instance")
	}

	accts := Wallet.GetAccounts()
	if len(accts) > 1 {
		return UnetworkRpc("error: does't support multi-addresses wallet locking asset")
	}

	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return UnetworkRpc("error: invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UnetworkRpc("error: invalid asset hash")
	}

	txn, err := sdk.MakeLockAssetTransaction(Wallet, assetID, value, uint32(height))
	if err != nil {
		return UnetworkRpc("error: " + err.Error())
	}

	txnHash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UnetworkRpc(errCode.Error())
	}
	return UnetworkRpc(BytesToHexString(txnHash.ToArrayReverse()))
}

func signMultisigTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UnetworkRpcNil
	}
	var signedrawtxn string
	switch params[0].(type) {
	case string:
		signedrawtxn = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}

	rawtxn, _ := HexStringToBytes(signedrawtxn)
	var txn tx.Transaction
	txn.Deserialize(bytes.NewReader(rawtxn))
	if len(txn.Programs) <= 0 {
		return UnetworkRpc("missing the first signature")
	}

	found := false
	programHashes := txn.ParseTransactionCode()
	for _, hash := range programHashes {
		acct := Wallet.GetAccountByProgramHash(hash)
		if acct != nil {
			found = true
			sig, _ := signature.SignBySigner(&txn, acct)
			txn.AppendNewSignature(sig)
		}
	}
	if !found {
		return UnetworkRpc("error: no available account detected")
	}

	_, needsig, err := txn.ParseTransactionSig()
	if err != nil {
		return UnetworkRpc("error: " + err.Error())
	}
	if needsig == 0 {
		txnHash := txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return UnetworkRpc(errCode.Error())
		}
		return UnetworkRpc(BytesToHexString(txnHash.ToArrayReverse()))
	} else {
		var buffer bytes.Buffer
		txn.Serialize(&buffer)
		return UnetworkRpc(BytesToHexString(buffer.Bytes()))
	}
}

func createMultisigTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 4 {
		return UnetworkRpcNil
	}
	var asset, from, address, value string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		from = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[3].(type) {
	case string:
		value = params[3].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return UnetworkRpc("error : wallet is not opened")
	}

	batchOut := sdk.BatchOut{
		Address: address,
		Value:   value,
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return UnetworkRpc("error: invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UnetworkRpc("error: invalid asset hash")
	}
	txn, err := sdk.MakeMultisigTransferTransaction(Wallet, assetID, from, batchOut)
	if err != nil {
		return UnetworkRpc("error: " + err.Error())
	}

	_, needsig, err := txn.ParseTransactionSig()
	if err != nil {
		return UnetworkRpc("error: " + err.Error())
	}
	if needsig == 0 {
		txnHash := txn.Hash()
		if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
			return UnetworkRpc(errCode.Error())
		}
		return UnetworkRpc(BytesToHexString(txnHash.ToArrayReverse()))
	} else {
		var buffer bytes.Buffer
		txn.Serialize(&buffer)
		return UnetworkRpc(BytesToHexString(buffer.Bytes()))
	}
}

func getBalance(params []interface{}) map[string]interface{} {
	if Wallet == nil {
		return UnetworkRpc("open wallet first")
	}
	type AssetInfo struct {
		AssetID string
		Value   string
	}
	balances := make(map[string][]*AssetInfo)
	accounts := Wallet.GetAccounts()
	coins := Wallet.GetCoins()
	for _, account := range accounts {
		assetList := []*AssetInfo{}
		programHash := account.ProgramHash
		for _, coin := range coins {
			if programHash == coin.Output.ProgramHash {
				var existed bool
				assetString := BytesToHexString(coin.Output.AssetID.ToArray())
				for _, info := range assetList {
					if info.AssetID == assetString {
						info.Value += coin.Output.Value.String()
						existed = true
						break
					}
				}
				if !existed {
					assetList = append(assetList, &AssetInfo{AssetID: assetString, Value: coin.Output.Value.String()})
				}
			}
		}
		address, _ := programHash.ToAddress()
		balances[address] = assetList
	}

	return UnetworkRpc(balances)
}

func registerUser(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UnetworkRpcNil
	}
	var userName, userProgramHash string
	switch params[0].(type) {
	case string:
		userName = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		userProgramHash = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	tmp, err := HexStringToBytesReverse(userProgramHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	programHash, err := Uint160ParseFromBytes(tmp)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	txn, err := sdk.MakeRegisterUserTransaction(userName, programHash)
	if err != nil {
		return UnetworkRpcInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UnetworkRpc(errCode.Error())
	}

	return UnetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

func postArticle(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UnetworkRpcNil
	}
	var articleHash, author string
	switch params[0].(type) {
	case string:
		articleHash = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		author = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}

	tmpHash, err := HexStringToBytes(articleHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	aHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	txn, err := sdk.MakePostArticleTransaction(Wallet, aHash, author)
	if err != nil {
		return UnetworkRpcInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UnetworkRpc(errCode.Error())
	}

	return UnetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

func replyArticle(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UnetworkRpcNil
	}
	var postTxnHash, contentHash, replier string
	var err error
	switch params[0].(type) {
	case string:
		postTxnHash = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		contentHash = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		replier = params[2].(string)
	default:
		return UnetworkRpcInvalidParameter
	}

	tmpHash, err := HexStringToBytesReverse(postTxnHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	pHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}

	tmpHash, err = HexStringToBytes(contentHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	cHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}

	txn, err := sdk.MakeReplyArticleTransaction(Wallet, pHash, cHash, replier)
	if err != nil {
		return UnetworkRpcInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UnetworkRpc(errCode.Error())
	}

	return UnetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

func likeArticle(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UnetworkRpcNil
	}
	var postTxnHash, liker string
	var likeType forum.LikeType
	switch params[0].(type) {
	case string:
		postTxnHash = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		liker = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		v, err := strconv.ParseInt(params[2].(string), 10, 8)
		if err != nil {
			return UnetworkRpcInvalidParameter
		}
		likeType = forum.LikeType(v)
	default:
		return UnetworkRpcInvalidParameter
	}

	tmpHash, err := HexStringToBytesReverse(postTxnHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	aHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	txn, err := sdk.MakeLikeArticleTransaction(Wallet, aHash, liker, likeType)
	if err != nil {
		return UnetworkRpcInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UnetworkRpc(errCode.Error())
	}

	return UnetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

func withdrawal(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UnetworkRpcNil
	}
	var payee, recipient, asset, amount string
	switch params[0].(type) {
	case string:
		payee = params[0].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		recipient = params[1].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		asset = params[2].(string)
	default:
		return UnetworkRpcInvalidParameter
	}
	switch params[3].(type) {
	case string:
		amount = params[3].(string)
	default:
		return UnetworkRpcInvalidParameter
	}

	tmpHash, err := HexStringToBytesReverse(recipient)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	aHash, err := Uint160ParseFromBytes(tmpHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}

	tmpHash, err = HexStringToBytesReverse(asset)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}
	bHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UnetworkRpcInvalidParameter
	}

	txn, err := sdk.MakeWithdrawalTransaction(Wallet, payee, aHash, bHash, amount)
	if err != nil {
		return UnetworkRpcInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UnetworkRpc(errCode.Error())
	}

	return UnetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}
