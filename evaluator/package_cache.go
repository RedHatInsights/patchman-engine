package evaluator

import (
	"app/base/database"
	"app/base/utils"
	"errors"
	"runtime"
	"time"

	rpm "github.com/ezamriy/gorpm"
	lru "github.com/hashicorp/golang-lru"
	"gorm.io/gorm"
)

var memoryPackageCache *PackageCache

const logProgressDuration = 5 * time.Second

type PackageCacheMetadata struct {
	ID              int64
	NameID          int64
	Name            string
	Evra            string
	SummaryHash     []byte
	DescriptionHash []byte
}

type PackageCache struct {
	enabled      bool
	preload      bool
	size         int
	nameSize     int
	byID         *lru.Cache
	byNevra      *lru.Cache
	latestByName *lru.Cache
	nameByID     *lru.Cache
}

func NewPackageCache(enabled bool, preload bool, size int, nameSize int) *PackageCache {
	c := new(PackageCache)

	c.enabled = enabled
	c.preload = preload
	c.size = size
	c.nameSize = nameSize

	if c.enabled {
		var err error
		c.byID, err = lru.New(c.size)
		if err != nil {
			panic(err)
		}
		c.byNevra, err = lru.New(c.size)
		if err != nil {
			panic(err)
		}
		c.latestByName, err = lru.New(c.nameSize)
		if err != nil {
			panic(err)
		}
		c.nameByID, err = lru.New(c.nameSize)
		if err != nil {
			panic(err)
		}
		return c
	}
	return nil
}

func (c *PackageCache) Load() {
	if !c.enabled || !c.preload {
		return
	}

	utils.Log("size", c.size).Info("PackageCache.Load")
	tx := database.Db.Begin()
	defer tx.Rollback()

	// load N last recently added packages, i.e. newest
	rows, err := tx.Table("package p").
		Select("p.id, p.name_id, pn.name, p.evra, p.summary_hash, p.description_hash").
		Joins("JOIN package_name pn ON pn.id = p.name_id").
		Order("id DESC").
		Limit(c.size).
		Rows()
	if err != nil {
		panic(err)
	}

	var mStart, mEnd runtime.MemStats
	runtime.ReadMemStats(&mStart)
	tStart := time.Now()

	progressTicker, count := utils.LogProgress("PackageCache.Load", logProgressDuration, int64(c.size))
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
		c.Add(&pkg)
		*count++
	}
	progressTicker.Stop()

	runtime.ReadMemStats(&mEnd)
	utils.Log("rows", c.byID.Len(), "allocated-size", utils.SizeStr(mEnd.TotalAlloc-mStart.TotalAlloc),
		"duration", utils.SinceStr(tStart, time.Millisecond)).Info("PackageCache.Load")
}

func (c *PackageCache) GetByID(id int64) (*PackageCacheMetadata, bool) {
	if c.enabled {
		val, ok := c.byID.Get(id)
		if ok {
			utils.Log("id", id).Trace("PackageCache.GetByID cache hit")
			metadata := val.(*PackageCacheMetadata)
			return metadata, true
		}
	}

	metadata := c.ReadByID(id)
	if c.enabled && metadata != nil {
		c.Add(metadata)
		utils.Log("id", id).Trace("PackageCache.GetByID read from db")
		return metadata, true
	}
	utils.Log("id", id).Trace("PackageCache.GetByID not found")
	return nil, false
}

func (c *PackageCache) GetByNevra(nevra string) (*PackageCacheMetadata, bool) {
	if c.enabled {
		val, ok := c.byNevra.Get(nevra)
		if ok {
			utils.Log("nevra", nevra).Trace("PackageCache.GetByNevra cache hit")
			metadata := val.(*PackageCacheMetadata)
			return metadata, true
		}
	}

	metadata := c.ReadByNevra(nevra)
	if c.enabled && metadata != nil {
		c.Add(metadata)
		utils.Log("nevra", nevra).Trace("PackageCache.GetByNevra read from db")
		return metadata, true
	}
	utils.Log("nevra", nevra).Trace("PackageCache.GetByNevra not found")
	return nil, false
}

func (c *PackageCache) GetLatestByName(name string) (*PackageCacheMetadata, bool) {
	if c.enabled {
		val, ok := c.latestByName.Get(name)
		if ok {
			utils.Log("name", name).Trace("PackageCache.GetLatestByName cache hit")
			metadata := val.(*PackageCacheMetadata)
			return metadata, true
		}
	}

	metadata := c.ReadLatestByName(name)
	if c.enabled && metadata != nil {
		c.Add(metadata)
		utils.Log("name", name).Trace("PackageCache.GetLatestByName read from db")
		return metadata, true
	}
	utils.Log("name", name).Trace("PackageCache.GetLatestByName not found")
	return nil, false
}

func (c *PackageCache) GetNameByID(id int64) (string, bool) {
	if c.enabled {
		val, ok := c.nameByID.Get(id)
		if ok {
			utils.Log("id", id).Trace("PackageCache.GetNameByID cache hit")
			metadata := val.(*PackageCacheMetadata)
			return metadata.Name, true
		}
	}

	metadata := c.ReadNameByID(id)
	if c.enabled && metadata != nil {
		c.Add(metadata)
		utils.Log("id", id).Trace("PackageCache.GetNameByID read from db")
		return metadata.Name, true
	}
	utils.Log("id", id).Trace("PackageCache.GetNameByID not found")
	return "", false
}

func (c *PackageCache) Add(pkg *PackageCacheMetadata) {
	c.addByID(pkg)
	c.addByNevra(pkg)
	c.addLatestByName(pkg)
	c.addNameByID(pkg)
}

func (c *PackageCache) addByID(pkg *PackageCacheMetadata) {
	evicted := c.byID.Add(pkg.ID, pkg)
	utils.Log("byID", pkg.ID, "evicted", evicted).Trace("PackageCache.addByID")
}

func (c *PackageCache) addByNevra(pkg *PackageCacheMetadata) {
	// make sure nevra contains epoch even if epoch==0
	nevra, err := utils.ParseNameEVRA(pkg.Name, pkg.Evra)
	if err != nil {
		utils.Log("id", pkg.ID, "name_id", pkg.NameID, "name", pkg.Name, "evra", pkg.Evra).
			Warn("PackageCache.addByNevra: cannot parse evra")
		return
	}
	nevraString := nevra.StringE(true)
	evicted := c.byNevra.Add(nevraString, pkg)
	utils.Log("byNevra", nevraString, "evicted", evicted).Trace("PackageCache.addByNevra")
}

func (c *PackageCache) addLatestByName(pkg *PackageCacheMetadata) {
	var latestEvra string
	latest, ok := c.latestByName.Peek(pkg.Name)
	if ok {
		latestEvra = latest.(*PackageCacheMetadata).Evra
	}
	if !ok || rpm.Vercmp(pkg.Evra, latestEvra) > 0 {
		// if there is no record yet
		// or it has older EVR we have to replace it
		evicted := c.latestByName.Add(pkg.Name, pkg)
		utils.Log("latestByName", pkg.Name, "evicted", evicted).Trace("PackageCache.addLatestByName")
	}
}

func (c *PackageCache) addNameByID(pkg *PackageCacheMetadata) {
	ok, evicted := c.nameByID.ContainsOrAdd(pkg.NameID, pkg)
	if !ok {
		// name was not there and we've added it
		utils.Log("nameByID", pkg.NameID, "evicted", evicted).Trace("PackageCache.addNameByID")
	}
}

func readPackageFromDB(where string, order string, args ...interface{}) *PackageCacheMetadata {
	tx := database.Db.Begin()
	defer tx.Rollback()

	var pkg PackageCacheMetadata
	query := tx.Table("package p").
		Select("p.id, p.name_id, pn.name, p.evra, p.summary_hash, p.description_hash").
		Joins("JOIN package_name pn ON pn.id = p.name_id").
		Where(where, args...)
	if order != "" {
		query = query.Order(order)
	}
	err := query.Take(&pkg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		panic(err)
	}
	return &pkg
}

func (c *PackageCache) ReadByID(id int64) *PackageCacheMetadata {
	return readPackageFromDB("p.id = ?", "", id)
}

func (c *PackageCache) ReadByNevra(nevraString string) *PackageCacheMetadata {
	nevra, err := utils.ParseNevra(nevraString)
	if err != nil {
		utils.Log("nevra", nevraString).Warn("PackageCache.ReadByNevra: cannot parse evra")
		return nil
	}
	utils.Log("nevra.Name", nevra.Name, "nevra.EVRAString", nevra.EVRAString()).Trace("PackageCache.ReadByNevra")
	return readPackageFromDB("pn.name = ? and p.evra = ?", "", nevra.Name, nevra.EVRAString())
}

func (c *PackageCache) ReadLatestByName(name string) *PackageCacheMetadata {
	return readPackageFromDB("pn.name = ?", "p.evra DESC", name)
}

func (c *PackageCache) ReadNameByID(id int64) *PackageCacheMetadata {
	return readPackageFromDB("pn.id = ?", "p.evra DESC", id)
}
