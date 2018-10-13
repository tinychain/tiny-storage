package types

import (
	"github.com/tinychain/tinychain/core/types"
)

type TransactionAPI interface {
	Call(tx *types.Transaction) ([]byte, error)
}

type ChainAPI interface {
}
