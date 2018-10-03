package rw

import (
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/tinychain/tinychain/common"
	"time"
)

var (
	defaultCollectTimeout = 60 * time.Second  // timeout for collecting copy messages
	defaultFetchTimeout   = 300 * time.Second // timeout for fetching data from IPFS
)

type Config struct {
	collectTimeout time.Duration // timeout for collecting copy message at a round
	fetchTimeout   time.Duration // timeout for fetching data from IPFS
	privKey        crypto.PrivKey
}

func newConfig(config *common.Config) (*Config, error) {
	conf := &Config{}
	conf.collectTimeout = config.GetDuration("storage.collect_timeout")
	if conf.collectTimeout == 0 {
		conf.collectTimeout = defaultCollectTimeout
	}

	conf.fetchTimeout = config.GetDuration("storage.fetch_timeout")
	if conf.fetchTimeout == 0 {
		conf.fetchTimeout = defaultFetchTimeout
	}

	privKey := config.GetString("storage.private_key")
	if privKey == "" {
		return conf, nil
	}

	privateKey, err := crypto.UnmarshalPrivateKey(common.Hex2Bytes(privKey))
	if err != nil {
		return nil, err
	}
	conf.privKey = privateKey
	return conf, nil
}
