package main

import (
	"fmt"
)

func StoreCode(code string, location int64, member_id int) bool {
	key := fmt.Sprintf("code:%d:%d", location, member_id)
	already_set := db.Exists(key).Val() == 1
	db.Set(key, code, 0)

	return already_set
}

func GetCode(location int64, member_id int) string {
	key := fmt.Sprintf("code:%d:%d", location, member_id)
	return db.Get(key).Val()
}
