package resolvers

import "github.com/astralmaster/elrond-go/storage"

func createBaseStorageResolver(
	storer storage.Storer,
	isFullHistoryNode bool,
) baseStorageResolver {
	if isFullHistoryNode {
		return &baseFullHistoryResolver{
			storer: storer,
		}
	}

	return &baseSimpleResolver{
		storer: storer,
	}
}
