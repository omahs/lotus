package modules

import (
	"context"
	"path/filepath"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"github.com/ipfs/go-filestore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	"github.com/ipfs/go-merkledag"
	"go.uber.org/fx"

	"github.com/filecoin-project/go-lotus/node/modules/dtypes"
	"github.com/filecoin-project/go-lotus/node/repo"
)

func Datastore(r repo.LockedRepo) (dtypes.MetadataDS, error) {
	return r.Datastore("/metadata")
}

func Blockstore(r repo.LockedRepo) (dtypes.ChainBlockstore, error) {
	blocks, err := r.Datastore("/blocks")
	if err != nil {
		return nil, err
	}

	bs := blockstore.NewBlockstore(blocks)
	return blockstore.NewIdStore(bs), nil
}

func ClientFstore(r repo.LockedRepo) (dtypes.ClientFilestore, error) {
	clientds, err := r.Datastore("/client")
	if err != nil {
		return nil, err
	}
	blocks := namespace.Wrap(clientds, datastore.NewKey("blocks"))

	fm := filestore.NewFileManager(clientds, filepath.Dir(r.Path()))
	fm.AllowFiles = true
	// TODO: fm.AllowUrls (needs more code in client import)

	bs := blockstore.NewBlockstore(blocks)
	return filestore.NewFilestore(bs, fm), nil
}

func ClientDAG(lc fx.Lifecycle, fstore dtypes.ClientFilestore) dtypes.ClientDAG {
	ibs := blockstore.NewIdStore((*filestore.Filestore)(fstore))
	bsvc := blockservice.New(ibs, offline.Exchange(ibs))
	dag := merkledag.NewDAGService(bsvc)

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return bsvc.Close()
		},
	})

	return dag
}

func ChainGCBlockstore(bs dtypes.ChainBlockstore, gcl dtypes.ChainGCLocker) dtypes.ChainGCBlockstore {
	return blockstore.NewGCBlockstore(bs, gcl)
}

func ChainBlockservice(bs dtypes.ChainBlockstore, rem dtypes.ChainExchange) dtypes.ChainBlockService {
	return blockservice.New(bs, rem)
}