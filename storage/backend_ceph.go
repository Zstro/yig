// +build ceph

package storage

import (
	"github.com/journeymidnight/yig/backend"
	"github.com/journeymidnight/yig/ceph"
	"github.com/journeymidnight/yig/crypto"
	"github.com/journeymidnight/yig/helper"
	"github.com/journeymidnight/yig/log"
	"github.com/journeymidnight/yig/meta"
	"github.com/journeymidnight/yig/meta/types"
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

	yig.DataStorage = ceph.Initialize(logger, helper.CONFIG)
	if len(yig.DataStorage) == 0 {
		helper.Logger.Panic(0, "PANIC: No data storage can be used!")
	}

	initializeRecycler(&yig)
	return &yig
}

func (yig *YigStorage) pickClusterAndPool(bucket string, object string,
	size int64, isAppend bool) (cluster backend.Cluster, poolName string) {

	metaClusters, _ := yig.MetaStorage.GetClusters()
	weights := make(map[string]int)
	for clusterID, cluster := range metaClusters {
		weights[clusterID] = cluster.Weight
	}

	objectType := types.ObjectTypeNormal
	if isAppend {
		objectType = types.ObjectTypeAppendable
	}
	cluster, poolName, _ = ceph.PickCluster(yig.DataStorage, weights, uint64(size),
		types.ObjectStorageClassStandard, objectType)
	return cluster, poolName
}
