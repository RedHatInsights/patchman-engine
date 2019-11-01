package listener

import (
	"gin-container/app/database"
	"gin-container/app/structures"
)

type Storage struct {
	buffer  *[]structures.HostDAO
}

func InitStorage(bufferSize int) *Storage{
	buffer := make([]structures.HostDAO, 0, bufferSize) // init empty array with given capacity
	storage := Storage{buffer: &buffer}
	return &storage
}

func (s *Storage) Add(host *structures.HostDAO) error {
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
	for _, item := range *s.buffer {
		err := database.Db.Save(&item).Error
		if err != nil {
			return err
		}
	}
	s.clean()
	return nil
}
