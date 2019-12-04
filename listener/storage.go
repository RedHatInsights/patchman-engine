package listener

import (
	"app/base/database"
	"app/base/structures"
	"app/base/utils"
	"fmt"
	"strconv"
	"strings"
)

type Storage struct {
	buffer        *[]structures.RhAccountDAO
	useBatchWrite bool
}

func InitStorage(bufferSize int, useBatchWrite bool) *Storage{
	buffer := make([]structures.RhAccountDAO, 0, bufferSize) // init empty array with given capacity
	storage := Storage{buffer: &buffer, useBatchWrite: useBatchWrite}
	utils.Log("useBatchWrite", useBatchWrite).Info("buffered storage created")
	return &storage
}

func (s *Storage) Add(host *structures.RhAccountDAO) error {
	if s.Capacity() == s.StoredItems() {
		err := s.Flush()
		if err != nil {
			return err
		}
	}
	*s.buffer = append(*s.buffer, *host)
	return nil
}

func (s *Storage) StoredItems() int {
	return len(*s.buffer)
}

func (s *Storage) Capacity() int {
	return cap(*s.buffer)
}

func (s *Storage) clean() {
	*s.buffer = (*s.buffer)[:0]
}

func (s *Storage) Flush() error {
	if s.useBatchWrite {
		err := s.flushBatch()
		return err
	} else {
		err := s.flushSimple()
		return err
	}
}

// slower solution working with both PostgreSQL and SQLite
func (s *Storage) flushSimple() error {
	for _, item := range *s.buffer {
		err := database.Db.Save(&item).Error
		if err != nil {
			return err
		}
	}
	s.clean()
	return nil
}

// https://stackoverflow.com/questions/12486436/how-do-i-batch-sql-statements-with-package-database-sql
func replaceSQL(stmt, pattern string, len int) string {
    pattern += ","
    stmt = fmt.Sprintf(stmt, strings.Repeat(pattern, len))
    n := 0
    for strings.IndexByte(stmt, '?') != -1 {
        n++
        param := "$" + strconv.Itoa(n)
        stmt = strings.Replace(stmt, "?", param, 1)
    }
    return strings.TrimSuffix(stmt, ",")
}

// use writing into the database in batches, doesn't work with SQLite
func (s *Storage) flushBatch() error {
	var vals []interface{}
	for _, item := range *s.buffer  {
		vals = append(vals, item.ID)
	}

	smt := `INSERT INTO hosts(id) VALUES %s`
	smt = replaceSQL(smt, "(?)", len(*s.buffer))
	tx, err := database.Db.DB().Begin()

	if err != nil {
		return err
	}

	_, err = tx.Exec(smt, vals...)
	if err != nil {
		errRb := tx.Rollback()
		if errRb != nil {
			return errRb
		}
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
