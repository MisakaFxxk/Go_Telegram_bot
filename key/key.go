package key

import (
	"database/sql"
	"log"
)

func Key_check(key string, db *sql.DB) (int, string) {
	check := 0
	var months string

	//查询Key是否存在
	key_exists, _ := Key_exists(key, db)
	if key_exists {
		// 查询数据库中的数据
		rows, err := db.Prepare("SELECT * FROM lite_keys WHERE lite_key = ?")
		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()

		var used_by_chatid, used_date sql.NullString
		var lite_key string
		var status int
		err = rows.QueryRow(key).Scan(&lite_key, &months, &status, &used_by_chatid, &used_date)
		if err != nil {
			panic(err.Error())
		}

		if status == 0 {
			check = 3 //key存在但已被使用
		} else if status == 1 {
			check = 1 //key存在且未被使用
		}
	} else {
		check = 2 //key不存在
	}

	return check, months
}

func Key_delete(key string, chatid int64, db *sql.DB) int {
	check := 0

	stmt, err := db.Prepare("UPDATE lite_keys set status = 0,used_by_chatid = ?,used_date = DATE(NOW()) where lite_key = ? ")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(chatid, key)
	if err != nil {
		log.Fatal(err)
	} else {
		check = 1
	}

	return check
}

func Key_exists(key string, db *sql.DB) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM lite_keys WHERE lite_key = ?)", key).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
