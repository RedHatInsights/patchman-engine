package evaluator

import (
	"app/base/database"
	"app/base/utils"
	"runtime"
	"sync"
	"time"
)

var memoryPackageCache *PackageCache

type PackageCacheMetadata struct {
	ID              int
	NameID          int
	Name            string
	Evra            string
	SummaryHash     []byte
	DescriptionHash []byte
}

type PackageCache struct {
	mtx          sync.RWMutex
	byID         map[int]*PackageCacheMetadata
	byNevra      map[string]*PackageCacheMetadata
	latestByName map[string]*PackageCacheMetadata
	nameByID     map[int]string
}

func NewPackageCache() *PackageCache {
	c := new(PackageCache)
	c.byID = map[int]*PackageCacheMetadata{}
	c.byNevra = map[string]*PackageCacheMetadata{}
	c.latestByName = map[string]*PackageCacheMetadata{}
	c.nameByID = map[int]string{}
	return c
}

func (c *PackageCache) Load() {
	tx := database.Db.Begin()
	defer tx.Rollback()
	rows, err := tx.Table("package p").
		Select("p.id, p.name_id, pn.name, p.evra, p.summary_hash, p.description_hash").
		Joins("JOIN package_name pn ON pn.id = p.name_id").
		Order("evra DESC").
		Rows()
	if err != nil {
		panic(err)
	}

	var mStart, mEnd runtime.MemStats
	runtime.ReadMemStats(&mStart)
	tStart := time.Now()

	c.mtx = sync.RWMutex{}
	c.mtx.Lock()
	defer c.mtx.Unlock()

	var columns PackageCacheMetadata
	for rows.Next() {
		err = tx.ScanRows(rows, &columns)
		if err != nil {
			panic(err)
		}
		pkg := PackageCacheMetadata{
			ID:              columns.ID,
			NameID:          columns.NameID,
			Name:            columns.Name,
			Evra:            columns.Evra,
			DescriptionHash: columns.DescriptionHash,
			SummaryHash:     columns.SummaryHash,
		}
		c.AddWithoutLock(&pkg)
	}
	runtime.ReadMemStats(&mEnd)
	utils.Log("rows", len(c.byID), "allocated-size", utils.SizeStr(mEnd.TotalAlloc-mStart.TotalAlloc),
		"duration", utils.SinceStr(tStart, time.Millisecond)).Info("PackageCache.Load")
}

func (c *PackageCache) GetByID(id int) (*PackageCacheMetadata, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	metadata, ok := c.byID[id]
	utils.Log("id", id, "ok", ok).Debug("PackageCache.GetByID")
	if ok {
		return metadata, true
	}
	return nil, false
}

func (c *PackageCache) GetByNevra(nevra string) (*PackageCacheMetadata, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	metadata, ok := c.byNevra[nevra]
	utils.Log("nevra", nevra, "ok", ok).Debug("PackageCache.GetByNevra")
	if ok {
		return metadata, true
	}
	return nil, false
}

func (c *PackageCache) GetLatestByName(name string) (*PackageCacheMetadata, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	metadata, ok := c.latestByName[name]
	utils.Log("name", name, "ok", ok).Debug("PackageCache.GetNameByID")
	if ok {
		return metadata, true
	}
	return nil, false
}

func (c *PackageCache) GetNameByID(id int) (string, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	metadata, ok := c.nameByID[id]
	utils.Log("id", id, "ok", ok).Debug("PackageCache.GetNameByID")
	return metadata, ok
}

func (c *PackageCache) Add(pkg *PackageCacheMetadata) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.AddWithoutLock(pkg)
}

func (c *PackageCache) AddWithoutLock(pkg *PackageCacheMetadata) {
	c.byID[pkg.ID] = pkg
	nevra := pkg.Name + "-" + pkg.Evra
	c.byNevra[nevra] = pkg
	if _, ok := c.latestByName[pkg.Name]; !ok {
		c.latestByName[pkg.Name] = pkg
	}
	if _, ok := c.nameByID[pkg.NameID]; !ok {
		c.nameByID[pkg.NameID] = pkg.Name
	}
	utils.Log("nevra", nevra).Debug("PackageCache.Add")
}
