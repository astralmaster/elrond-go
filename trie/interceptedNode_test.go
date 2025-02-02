package trie_test

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/trie"
	"github.com/stretchr/testify/assert"
)

func getDefaultInterceptedTrieNodeParameters(t *testing.T) ([]byte, marshal.Marshalizer, hashing.Hasher) {
	tr := initTrie(t)
	nodes, _ := getEncodedTrieNodesAndHashes(tr)

	return nodes[0], &testscommon.ProtobufMarshalizerMock{}, &testscommon.KeccakMock{}
}

func getEncodedTrieNodesAndHashes(tr common.Trie) ([][]byte, [][]byte) {
	it, _ := trie.NewIterator(tr)
	encNode, _ := it.MarshalizedNode()

	nodes := make([][]byte, 0)
	nodes = append(nodes, encNode)

	hashes := make([][]byte, 0)
	hash, _ := it.GetHash()
	hashes = append(hashes, hash)

	for it.HasNext() {
		_ = it.Next()
		encNode, _ = it.MarshalizedNode()

		nodes = append(nodes, encNode)
		hash, _ = it.GetHash()
		hashes = append(hashes, hash)
	}

	return nodes, hashes
}

func TestNewInterceptedTrieNode_EmptyBufferShouldFail(t *testing.T) {
	t.Parallel()

	_, marsh, hasher := getDefaultInterceptedTrieNodeParameters(t)
	interceptedNode, err := trie.NewInterceptedTrieNode([]byte{}, marsh, hasher)
	assert.True(t, check.IfNil(interceptedNode))
	assert.Equal(t, trie.ErrValueTooShort, err)
}

func TestNewInterceptedTrieNode_NilMarshalizerShouldFail(t *testing.T) {
	t.Parallel()

	buff, _, hasher := getDefaultInterceptedTrieNodeParameters(t)
	interceptedNode, err := trie.NewInterceptedTrieNode(buff, nil, hasher)
	assert.True(t, check.IfNil(interceptedNode))
	assert.Equal(t, trie.ErrNilMarshalizer, err)
}

func TestNewInterceptedTrieNode_NilHasherShouldFail(t *testing.T) {
	t.Parallel()

	buff, marsh, _ := getDefaultInterceptedTrieNodeParameters(t)
	interceptedNode, err := trie.NewInterceptedTrieNode(buff, marsh, nil)
	assert.True(t, check.IfNil(interceptedNode))
	assert.Equal(t, trie.ErrNilHasher, err)
}

func TestNewInterceptedTrieNode_OkParametersShouldWork(t *testing.T) {
	t.Parallel()

	interceptedNode, err := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.False(t, check.IfNil(interceptedNode))
	assert.Nil(t, err)
}

func TestInterceptedTrieNode_CheckValidity(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))

	err := interceptedNode.CheckValidity()
	assert.Nil(t, err)
}

func TestInterceptedTrieNode_Hash(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	tr := initTrie(t)
	_, hashes := getEncodedTrieNodesAndHashes(tr)

	hash := interceptedNode.Hash()
	assert.Equal(t, hashes[0], hash)
}

func TestInterceptedTrieNode_GetSerialized(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	tr := initTrie(t)
	nodes, _ := getEncodedTrieNodesAndHashes(tr)

	encNode := interceptedNode.GetSerialized()
	assert.Equal(t, nodes[0], encNode)
}

func TestInterceptedTrieNode_SetSerialized(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	serializedNode := []byte("serialized node")

	interceptedNode.SetSerialized(serializedNode)
	assert.Equal(t, serializedNode, interceptedNode.GetSerialized())
}

func TestInterceptedTrieNode_IsForCurrentShard(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.True(t, interceptedNode.IsForCurrentShard())
}

func TestInterceptedTrieNode_Type(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.Equal(t, "intercepted trie node", interceptedNode.Type())
}

func TestInterceptedTrieNode_String(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.NotEqual(t, 0, interceptedNode.String())
}

func TestInterceptedTrieNode_SenderShardId(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.NotEqual(t, 0, interceptedNode.SenderShardId())
}

func TestInterceptedTrieNode_ReceiverShardId(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.NotEqual(t, 0, interceptedNode.ReceiverShardId())
}

func TestInterceptedTrieNode_Nonce(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.NotEqual(t, 0, interceptedNode.Nonce())
}

func TestInterceptedTrieNode_SenderAddress(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.Nil(t, interceptedNode.SenderAddress())
}

func TestInterceptedTrieNode_Fee(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.Equal(t, big.NewInt(0), interceptedNode.Fee())
}

func TestInterceptedTrieNode_Identifiers(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.Equal(t, [][]byte{interceptedNode.Hash()}, interceptedNode.Identifiers())
}

func TestInterceptedTrieNode_SizeInBytes(t *testing.T) {
	t.Parallel()

	interceptedNode, _ := trie.NewInterceptedTrieNode(getDefaultInterceptedTrieNodeParameters(t))
	assert.Equal(t, 380, interceptedNode.SizeInBytes())
}
