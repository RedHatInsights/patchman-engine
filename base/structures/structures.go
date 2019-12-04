package structures

type RhAccountDAO struct {
	ID              int
	Name            string
}

// db table name, for gorm
func (RhAccountDAO) TableName() string {
	return "rh_account"
}
