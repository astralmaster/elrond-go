package facade

import (
	"github.com/astralmaster/elrond-go/ntp"
)

// GetSyncer returns the current syncer
func (nf *nodeFacade) GetSyncer() ntp.SyncTimer {
	return nf.syncer
}
