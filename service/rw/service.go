package rw

import (
	"github.com/tinychain/tinychain/p2p/pb"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/op/go-logging"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tiny-storage/types"
	"time"
)

var (
	rwServiceMsg = "storage.rw"
)

// RWService implements the service of reading and writing data to IPFS, with the specific expected amount of
// store peers `n` and storing duration `t`.
// The service makes sure the storing request will be transfer to at least `n` peers, but does not guarantee the
// real amount of peers storing a copy of data is `n`.
//
// Before using RWService, the `rw_manager` contract should be deployed in blockchain.
type RWService struct {
	log            *logging.Logger
	collectTimeout time.Duration // timeout for collecting copy message at a round
}

func NewRWService() *RWService {
	rw := &RWService{
		log: common.GetLogger("storage_rw_service"),
	}
}

// process handles copy message from other peers.
func (rw *RWService) process(copyMsg *types.CopyMessage) error {
	// Verify sign list
}

func (rw *RWService) Run(pid peer.ID, message *pb.Message) error {
	data := message.Data
	copyMsg := &types.CopyMessage{}
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
