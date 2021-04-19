package evaluator

import (
	"app/base/database"
	"sync"
)

var packageNameCacheData *packageNameCache

type PackageMetadata struct {
	NameID          int
	SummaryHash     []byte
	DescriptionHash []byte
}

type packageNameCache struct {
	mtx  sync.RWMutex
	data map[string]PackageMetadata
}

type packageMetadataModel struct {
	Name string
	PackageMetadata
}

func ConfigurePackageNameCache() {
	tx := database.Db.Begin()
	defer tx.RollbackUnlessCommitted()
	rows, err := tx.Table("package").
		Select("DISTINCT ON (name_id) name_id, name, summary_hash, description_hash").
		Joins("JOIN package_name pn ON pn.id = name_id").
		Where("summary_hash IS NOT NULL").
		Where("description_hash IS NOT NULL").
		Order("name_id, evra DESC").Rows()
	if err != nil {
		panic(err)
	}

	cache := packageNameCache{}
	cache.mtx = sync.RWMutex{}
	cache.mtx.Lock()
	defer cache.mtx.Unlock()
	cache.data = map[string]PackageMetadata{}
	var model packageMetadataModel
	for rows.Next() {
		err = tx.ScanRows(rows, &model)
		if err != nil {
			panic(err)
		}
		cache.data[model.Name] = PackageMetadata{
			NameID:          model.NameID,
			DescriptionHash: model.DescriptionHash,
			SummaryHash:     model.SummaryHash,
		}
	}
	packageNameCacheData = &cache
}

func GetPackageNameMetadata(name string) (*PackageMetadata, bool) {
	packageNameCacheData.mtx.RLock()
	defer packageNameCacheData.mtx.RUnlock()
	metadata, ok := packageNameCacheData.data[name]
	if ok {
		return &metadata, true
	}
	return nil, false
}
