// +build seaweedfs

package storage

import (
	"github.com/journeymidnight/yig/backend"
	"github.com/journeymidnight/yig/crypto"
	"github.com/journeymidnight/yig/helper"
	"github.com/journeymidnight/yig/log"
	"github.com/journeymidnight/yig/meta"
	"github.com/journeymidnight/yig/seaweed"
	"sync"
)

func New(logger *log.Logger, metaCacheType int, enableDataCache bool) *YigStorage {
	kms := crypto.NewKMS()
	yig := YigStorage{
		DataStorage: make(map[string]backend.Cluster),
		DataCache:   newDataCache(enableDataCache),
		MetaStorage: meta.New(logger, meta.CacheType(metaCacheType)),
		KMS:         kms,
		Logger:      logger,
		Stopping:    false,
		WaitGroup:   new(sync.WaitGroup),
	}

	yig.DataStorage = seaweed.Initialize(logger, helper.CONFIG)
	if len(yig.DataStorage) == 0 {
		helper.Logger.Panic(0, "PANIC: No data storage can be used!")
	}

	initializeRecycler(&yig)
	return &yig
}

func (yig *YigStorage) pickClusterAndPool(bucket string, object string,
	size int64, isAppend bool) (cluster backend.Cluster, poolName string) {
	// TODO cluster picking logic
	cluster, poolName, _ = seaweed.PickCluster(yig.DataStorage,
		nil, uint64(size), 0, 0)
	return cluster, poolName
}
