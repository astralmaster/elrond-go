package peer

import (
	"github.com/astralmaster/elrond-go/dataRetriever"
)

// DataPool indicates the main functionality needed in order to fetch the required blocks from the pool
type DataPool interface {
	Headers() dataRetriever.HeadersPool
	IsInterfaceNil() bool
}
