package rw

import (
	"github.com/tinychain/tinychain/p2p/pb"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/op/go-logging"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tiny-storage/protocol"
	"github.com/tinychain/tinychain/core/vm/evm/abi"
	"time"
	"io/ioutil"
	"strings"
	"github.com/tinychain/tinychain/rpc/api"
	"github.com/tinychain/tinychain/core/types"
	"github.com/libp2p/go-libp2p-crypto"
	"errors"
	"sync/atomic"
)

var (
	errPrivKeyNotFound = errors.New("private key should be provided")

	rwServiceMsg = "storage.rw"

	defaultCollectTimeout = 60 * time.Second  // timeout for collecting copy messages
	defaultFetchTimeout   = 300 * time.Second // timeout for fetching data from IPFS
)

type account struct {
}

// RWService implements the service of reading and writing data to IPFS, with the specific expected amount of
// store peers `n` and storing duration `t`.
// The service makes sure the storing request will be transfer to at least `n` peers, but does not guarantee the
// real amount of peers storing a copy of data is `n`.
//
// Before using RWService, the `rw_manager` contract should be deployed in blockchain.
type RWService struct {
	log            *logging.Logger
	privKey        crypto.PrivKey // private key used to sign proof fetching transaction
	contractAddr   common.Address // address of pre-deploy contract `storage_rw.solc`
	collectTimeout time.Duration  // timeout for collecting copy message at a round
	fetchTimeout   time.Duration

	api     *api.TransactionAPI // transaction api provided by blockchain
	address atomic.Value
}

func NewRWService(config *common.Config, api *api.TransactionAPI, privKey crypto.PrivKey) *RWService {
	rw := &RWService{
		log:     common.GetLogger("storage_rw_service"),
		api:     api,
		privKey: privKey,
	}

	rw.collectTimeout = config.GetDuration("storage.collect_timeout")
	if rw.collectTimeout == 0 {
		rw.collectTimeout = defaultCollectTimeout
	}

	rw.fetchTimeout = config.GetDuration("storage.fetch_timeout")
	if rw.fetchTimeout == 0 {
		rw.fetchTimeout = defaultFetchTimeout
	}

	return rw
}

func (rw *RWService) Addr() common.Address {
	if addr := rw.address.Load(); addr != nil {
		return addr.(common.Address)
	}
	addr, err := common.GenAddrByPrivkey(rw.privKey)
	if err != nil {
		return common.Address{}
	}
	rw.address.Store(addr)
	return addr
}

// process handles copy message from other peers.
func (rw *RWService) process(copyMsg *protocol.CopyMessage) error {
	// Verify proof from blockchain
	payload, err := rw.pack("getProof", copyMsg.ProofId)
	if err != nil {
		rw.log.Errorf("pack function and params to payload failed, %s", err)
		return err
	}

	tx := types.NewTransaction(0, 0, 0, nil, payload, rw.Addr(), rw.contractAddr)
	ret, err := rw.api.Call(tx)
	if err != nil {
		rw.log.Errorf("error occurs when calling contract, %s", err)
		return err
	}

	// Verify copy message
	if err := copyMsg.Verify(); err != nil {
		rw.log.Errorf("copy message is not valid, %s", err)
		return err
	}


}

func (rw *RWService) pack(fn string, args ...interface{}) ([]byte, error) {
	jsonData, err := ioutil.ReadFile("./contracts/storage_rw.abi")
	if err != nil {
		return nil, err
	}
	abi, err := abi.JSON(strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	return abi.Pack(fn, args)
}

func (rw *RWService) Run(pid peer.ID, message *pb.Message) error {
	data := message.Data
	copyMsg := &protocol.CopyMessage{}
	if err := copyMsg.Deserialize(data); err != nil {
		rw.log.Error(err)
		return err
	}

	return rw.process(copyMsg)
}

func (rw *RWService) Type() string {
	return rwServiceMsg
}

func (rw *RWService) Error(err error) {
	rw.log.Error(err)
}
