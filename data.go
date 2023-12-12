package main

import (
	"fmt"
)

func StoreCode(code string, location int64, member_id int) bool {
	key := fmt.Sprintf("code:%d:%d", location, member_id)

	val_exists := db.Exists(key).Val()
	db.Set(key, code, 0)
	val := db.Get(code).Val()

	fmt.Println(val, val_exists)

	return false
}
