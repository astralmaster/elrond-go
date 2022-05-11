package spos

import (
	"github.com/astralmaster/elrond-go/common"
	"github.com/astralmaster/elrond-go/sharding"
)

// GetConsensusTopicID will construct and return the topic ID based on shard coordinator
func GetConsensusTopicID(shardCoordinator sharding.Coordinator) string {
	return common.ConsensusTopic + shardCoordinator.CommunicationIdentifier(shardCoordinator.SelfId())
}
