package datebese

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func NewDB() (*sql.DB, error) {
	//初始化数据库
	var err error
	db, err := sql.Open("mysql", "")
	if err != nil {
		log.Fatal(err)
	}

	// 检查连接是否成功
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("数据库连接成功")
	}

	return db, nil
}
