package detector

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/interceptedBlocks"
	"github.com/ElrondNetwork/elrond-go/process/slash"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type detectorCache interface {
	add(round uint64, pubKey []byte, data process.InterceptedData)
	data(round uint64, pubKey []byte) dataList
	validators(round uint64) [][]byte
}

// multipleHeaderProposalsDetector - checks slashable events in case a validator proposes multiple(possibly) malicious headers.
type multipleHeaderProposalsDetector struct {
	cache            detectorCache
	nodesCoordinator sharding.NodesCoordinator
	baseSlashingDetector
}

// NewMultipleHeaderProposalsDetector - creates a new multipleHeaderProposalsDetector for multiple headers
// proposal detection or multiple headers proposal proof verification
func NewMultipleHeaderProposalsDetector(
	nodesCoordinator sharding.NodesCoordinator,
	roundHandler process.RoundHandler,
	maxRoundCacheSize uint64,
) (slash.SlashingDetector, error) {
	if check.IfNil(nodesCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(roundHandler) {
		return nil, process.ErrNilRoundHandler
	}

	//TODO: Here, instead of CacheSize, use maxRoundCacheSize = from config file
	cache := newRoundProposerDataCache(CacheSize)
	baseDetector := baseSlashingDetector{roundHandler: roundHandler}

	return &multipleHeaderProposalsDetector{
		cache:                cache,
		nodesCoordinator:     nodesCoordinator,
		baseSlashingDetector: baseDetector,
	}, nil
}

// VerifyData - checks if an intercepted data(which should be a header) represents a slashable event.
// If another header with the same round and proposer exists, but a different hash, then a proof of type
// slash.MultipleProposal is provided, otherwise a nil proof, along with an error is provided indicating that
// no slashing event has been detected or an error occurred verifying the data.
func (mhp *multipleHeaderProposalsDetector) VerifyData(data process.InterceptedData) (slash.SlashingProofHandler, error) {
	header, castOk := data.(*interceptedBlocks.InterceptedHeader)
	if !castOk {
		return nil, process.ErrCannotCastInterceptedDataToHeader
	}

	round := header.HeaderHandler().GetRound()
	if !mhp.isRoundRelevant(round) {
		return nil, process.ErrHeaderRoundNotRelevant
	}

	proposer, err := mhp.getProposerPubKey(header.HeaderHandler())
	if err != nil {
		return nil, err
	}

	slashType, slashLevel, headers := mhp.getSlashingResult(header, round, proposer)
	mhp.cache.add(round, proposer, header)

	if slashType == slash.MultipleProposal {
		return slash.NewMultipleProposalProof(
			slash.SlashingData{
				SlashingLevel: slashLevel,
				Data:          headers,
			},
		)
	}
	return nil, process.ErrNoSlashingEventDetected
}

func (mhp *multipleHeaderProposalsDetector) getProposerPubKey(header data.HeaderHandler) ([]byte, error) {
	validators, err := mhp.nodesCoordinator.ComputeConsensusGroup(
		header.GetRandSeed(),
		header.GetRound(),
		header.GetShardID(),
		header.GetEpoch())

	if err != nil {
		return nil, err
	}
	if len(validators) == 0 {
		return nil, process.ErrEmptyConsensusGroup
	}

	return validators[0].PubKey(), nil
}

func (mhp *multipleHeaderProposalsDetector) getSlashingResult(
	currHeader process.InterceptedData,
	currRound uint64,
	proposerPubKey []byte,
) (slash.SlashingType, slash.SlashingLevel, []process.InterceptedData) {
	headers := make([]process.InterceptedData, 0)
	slashType := slash.None
	slashLevel := slash.Level0
	proposedHeaders := mhp.cache.data(currRound, proposerPubKey)

	if len(proposedHeaders) >= 1 {
		headers = mhp.getProposedHeadersWithDifferentHash(currHeader.Hash(), proposedHeaders)
		if len(headers) >= 1 {
			headers = append(headers, currHeader)
			slashLevel = computeSlashLevel(headers)
			slashType = slash.MultipleProposal
		}
	}

	return slashType, slashLevel, headers
}

func (mhp *multipleHeaderProposalsDetector) getProposedHeadersWithDifferentHash(currHash []byte, otherHeaders dataList) []process.InterceptedData {
	ret := make([]process.InterceptedData, 0)

	for _, currHeader := range otherHeaders {
		if !bytes.Equal(currHash, currHeader.Hash()) {
			ret = append(ret, currHeader)
		}
	}

	return ret
}

// ValidateProof - validates if the given proof is valid.
// For a proof of type slash.MultipleProposal to be valid, it should:
//  - Be of either level slash.Level1 (with 2 proposed headers) OR slash.Level2 (with >2 proposed headers)
//  - Have all proposed headers with the same round and proposer, but different hashes
func (mhp *multipleHeaderProposalsDetector) ValidateProof(proof slash.SlashingProofHandler) error {
	multipleProposalProof, castOk := proof.(slash.MultipleProposalProofHandler)
	if !castOk {
		return process.ErrCannotCastProofToMultipleProposedHeaders
	}
	if proof.GetType() != slash.MultipleProposal {
		return process.ErrInvalidSlashType
	}

	err := checkSlashLevel(multipleProposalProof.GetHeaders(), multipleProposalProof.GetLevel())
	if err != nil {
		return err
	}

	return mhp.checkProposedHeaders(multipleProposalProof.GetHeaders())
}

func (mhp *multipleHeaderProposalsDetector) checkProposedHeaders(headers []*interceptedBlocks.InterceptedHeader) error {
	if len(headers) < MinSlashableNoOfHeaders {
		return process.ErrNotEnoughHeadersProvided
	}

	hashes := make(map[string]struct{})
	round := headers[0].HeaderHandler().GetRound()
	proposer, err := mhp.getProposerPubKey(headers[0].HeaderHandler())
	if err != nil {
		return err
	}

	for _, header := range headers {
		hash := string(header.Hash())
		if _, exists := hashes[hash]; exists {
			return process.ErrHeadersShouldHaveDifferentHashes
		}

		err = mhp.checkHeaderHasSameProposerAndRound(header, round, proposer)
		if err != nil {
			return err
		}

		hashes[hash] = struct{}{}
	}

	return nil
}

func (mhp *multipleHeaderProposalsDetector) checkHeaderHasSameProposerAndRound(
	header *interceptedBlocks.InterceptedHeader,
	round uint64,
	proposer []byte,
) error {
	if header.HeaderHandler().GetRound() != round {
		return process.ErrHeadersDoNotHaveSameRound
	}

	currProposer, err := mhp.getProposerPubKey(header.HeaderHandler())
	if err != nil {
		return err
	}

	if !bytes.Equal(proposer, currProposer) {
		return process.ErrHeadersDoNotHaveSameProposer
	}

	return nil
}
