package evaluator

import "app/base/database"

var STATUS = make(map[int]string, 2)

const INSTALLABLE = 0
const APPLICABLE = 1

type statusRow struct {
	ID   int
	Name string
}

func configureStatus() {
	var rows []statusRow

	err := database.Db.Table("status s").Select("id, name").Scan(&rows).Error
	if err != nil {
		panic(err)
	}

	for _, r := range rows {
		STATUS[r.ID] = r.Name
	}
}
