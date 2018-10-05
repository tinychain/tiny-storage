package rw

import (
	"errors"
	"io/ioutil"
	"strings"
	"sync/atomic"

	"github.com/ipfs/go-ipfs-api"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/op/go-logging"
	"github.com/tinychain/tiny-storage/types"
	"github.com/tinychain/tinychain/common"
	bcTypes "github.com/tinychain/tinychain/core/types"
	"github.com/tinychain/tinychain/core/vm/evm/abi"
	"github.com/tinychain/tinychain/event"
	"github.com/tinychain/tinychain/p2p"
	"github.com/tinychain/tinychain/p2p/pb"
	"sync"
	"time"
	"github.com/hashicorp/golang-lru"
)

type period int

const (
	notCollect      period = iota
	collecting
	collectComplete

	lruMaxSize = 65535
)

var (
	errPrivKeyNotFound = errors.New("private key should be provided")

	rwServiceMsg = "storage.rw"
)

// RWService implements the service of reading and writing data to IPFS, with the specific expected amount of
// store peers `n` and storing duration `t`.
// The service makes sure the storing request will be transfer to at least `n` peers, but does not guarantee the
// real amount of peers storing a copy of data is `n`.
//
// Before using RWService, the `rw_manager` contract should be deployed in blockchain.
type RWService struct {
	log          *logging.Logger
	mu           sync.RWMutex
	conf         *Config
	contractAddr common.Address // address of pre-deploy contract `storage_rw.solc`

	api     types.TransactionAPI // transaction api provided by blockchain
	ipfs    shell.Shell          // ipfs shell handler, **there should be an ipfs daemon starting in local system**
	event   *event.TypeMux
	address atomic.Value

	msgCache    map[string]map[string]*types.CopyMessage // cache for collecting enough copy messages
	proofCache  *lru.Cache
	collectFlag map[string]period
}

func NewRWService(config *common.Config, api types.TransactionAPI) *RWService {
	conf, err := newConfig(config)
	if err != nil {
		return nil
	}
	rw := &RWService{
		log:         common.GetLogger("storage_rw_service"),
		conf:        conf,
		msgCache:    make(map[string]map[string]*types.CopyMessage),
		collectFlag: make(map[string]period),
		api:         api,
		event:       event.GetEventhub(),
	}

	rw.proofCache, err = lru.New(lruMaxSize)
	if err != nil {
		return nil
	}
	return rw
}

func (rw *RWService) Addr() common.Address {
	if addr := rw.address.Load(); addr != nil {
		return addr.(common.Address)
	}
	addr, err := common.GenAddrByPrivkey(rw.PrivKey())
	if err != nil {
		return common.Address{}
	}
	rw.address.Store(addr)
	return addr
}

func (rw *RWService) PrivKey() crypto.PrivKey {
	return rw.conf.privKey
}

// process handles copy message from other peers.
func (rw *RWService) process(copyMsg *types.CopyMessage) error {
	// Check there is private key or not
	if rw.PrivKey() == nil {
		// Only relay the copy message
		data, err := copyMsg.Serialize()
		if err != nil {
			rw.log.Errorf("encode copyMsg failed when only relay, %s", err)
			return err
		}
		go rw.event.Post(&p2p.MulticastNeighborEvent{
			Typ:   rwServiceMsg,
			Data:  data,
			Count: copyMsg.Peers,
		})
		return nil
	}
	// Verify proof from contracts `storage_rw.solc` which has deployed in blockchain
	payload, err := rw.pack("getProof", copyMsg.ProofId)
	if err != nil {
		rw.log.Errorf("pack function and params to payload failed, %s", err)
		return err
	}

	tx := bcTypes.NewTransaction(0, 0, 0, nil, payload, rw.Addr(), rw.contractAddr)
	ret, err := rw.api.Call(tx)
	if err != nil {
		rw.log.Errorf("error occurs when calling contract, %s", err)
		return err
	}

	var proof *types.Proof
	cache, ok := rw.proofCache.Get(copyMsg.ID())
	if ok {
		proof = cache.(*types.Proof)
	} else {
		proof = &types.Proof{}
		if err := proof.Deserialize(ret); err != nil {
			rw.log.Errorf("decode proof from bytes failed, %s", err)
			return err
		}
		rw.proofCache.Add(copyMsg.ID(), proof)
	}

	// Check is there copy in local
	if _, err := rw.GetFromLocal(proof.Cid); err == nil {
		// File exists in local storage
		rw.log.Infof("data with cid %s exists in local storage", proof.Cid)
		return nil
	}

	if err := rw.startCollect(copyMsg, proof); err != nil {
		rw.log.Errorf("error occurs when calling collect copy message, %s", err)
		return err
	}

	return nil
}

func (rw *RWService) verifyAndFetch(copyMsg *types.CopyMessage, proof *types.Proof) error {
	// Verify copy message with proof
	if err := copyMsg.Verify(proof); err != nil {
		rw.log.Errorf("copy message is not valid, %s", err)
		return err
	}

	// Fetch data from IPFS
	if _, err := rw.GetFromIPFS(proof.Cid); err != nil {
		rw.log.Errorf("fetch data from IPFS failed, %s", err)
		return err
	}

	// Verify info in fileDesc with real data from IPFS
	if err := rw.VerifyWithIPFS(copyMsg.FileDesc); err != nil {
		rw.log.Errorf("fileDesc invalid and mismatch with real data from IPFS, %s", err)
		rw.DeleteData(proof.Cid)
		return err
	}

	// Relay message to other peers
	copyMsg.Sign(rw.PrivKey())
	data, err := copyMsg.Serialize()
	if err != nil {
		return err
	}
	go rw.event.Post(&p2p.MulticastNeighborEvent{
		Typ:   rwServiceMsg,
		Data:  data,
		Count: copyMsg.Peers,
	})
	return nil
}

func (rw *RWService) startCollect(copyMsg *types.CopyMessage, proof *types.Proof) error {
	msgMap := rw.msgCache[copyMsg.ID()]
	if msgMap == nil {
		msgMap = make(map[string]*types.CopyMessage)
		rw.msgCache[copyMsg.ID()] = msgMap
	}
	msgHash, err := copyMsg.Hash()
	if err != nil {
		rw.log.Errorf("failed to get hash of copy message, %s", err)
		return err
	}
	rw.mu.RLock()
	flag := rw.collectFlag[copyMsg.ID()]
	rw.mu.RUnlock()
	if flag == collecting {
		if _, exist := msgMap[msgHash]; exist {
			return nil
		}
		msgMap[msgHash] = copyMsg
	} else if flag == collectComplete {
		return nil
	} else {
		go func() {
			timer := time.NewTimer(rw.conf.collectTimeout)
			<-timer.C
			rw.mu.Lock()
			// Mark collect period is completed
			rw.collectFlag[msgHash] = collectComplete
			rw.mu.Unlock()
			if err := rw.verifyAndFetch(rw.pickBestCopy(copyMsg.ID()), proof); err != nil {
				rw.log.Errorf("error occurs when verify and fetch data with IPFS, %s", err)
			}
			return
		}()
	}
	return nil
}

// pickBestCopy picks the copy message with longest sign list, which means spread the farthest.
func (rw *RWService) pickBestCopy(cid string) *types.CopyMessage {
	var (
		maxCount int
		tmsg     *types.CopyMessage
	)
	msgs := rw.msgCache[cid]
	for _, msg := range msgs {
		if len(msg.SignList) > maxCount {
			tmsg = msg
		}
	}

	return tmsg
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

// Run implements `protocol` interface provided by blockchain as network servcie.
func (rw *RWService) Run(pid peer.ID, message *pb.Message) error {
	data := message.Data
	copyMsg := &types.CopyMessage{}
	if err := copyMsg.Deserialize(data); err != nil {
		rw.log.Error(err)
		return err
	}

	return rw.process(copyMsg)
}

// Type implements `protocol` interface provided by blockchain as network servcie.
func (rw *RWService) Type() string {
	return rwServiceMsg
}

// Error implements `protocol` interface provided by blockchain as network servcie.
func (rw *RWService) Error(err error) {
	rw.log.Error(err)
}
