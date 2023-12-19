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

func RemoveCode(location int64, member_id int) {
	key := fmt.Sprintf("code:%d:%d", location, member_id)
	db.Del(key)
}

// SetCodeRequired sets whether or not a guest code is required for a given location.
// This acts as sort of a 'cache' to avoid testing guest code requirements every time.
func SetCodeRequired(location int64, required bool) {
	key := fmt.Sprintf("code_required:%d", location)
	db.Set(key, required, 0)
}

// GetCodeRequired returns whether or not a guest code is required for a given location.
// This uses the const values defined in types.go: GuestCodeRequired, GuestCodeNotRequired, and Unknown.
// In the case that no tests have been performed, Unknown will be returned.
func GetCodeRequired(location int64) uint {
	key := fmt.Sprintf("code_required:%d", location)
	if db.Exists(key).Val() == 0 {
		return Unknown
	}
	if db.Get(key).Val() == "true" {
		return GuestCodeRequired
	}
	return GuestCodeNotRequired
}
