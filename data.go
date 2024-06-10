package main

import (
	"fmt"

	"github.com/go-redis/redis"
)

// LocationExists checks if a location identifier is valid (as known by the cache).
// Cache rarely will change, so this is a good way to check if a location is valid.
func LocationExists(location int64) bool {
	_, ok := cachedLocationsMap[uint(location)]
	return ok
}

// StoreCode stores a guest code for a given location and member ID.
func StoreCode(code string, location int64, member_id int) bool {
	key := fmt.Sprintf("code:%d:%d", location, member_id)
	already_set := db.Exists(key).Val() == 1
	db.Set(key, code, 0)

	return already_set
}

// GetCode returns the guest code for a given location and member ID.
func GetCode(location int64, member_id int) (string, bool) {
	key := fmt.Sprintf("code:%d:%d", location, member_id)
	result := db.Get(key)
	return result.Val(), result.Err() == redis.Nil
}

// RemoveCode removes a guest code for a given location and member ID.
func RemoveCode(location int64, member_id int) {
	key := fmt.Sprintf("code:%d:%d", location, member_id)
	db.Del(key)
}

// SetCodeRequirement sets whether or not a guest code is required for a given location.
// This acts as sort of a 'cache' to avoid testing guest code requirements every time.
func SetCodeRequirement(location int64, required bool) {
	key := fmt.Sprintf("code_required:%d", location)
	db.Set(key, required, 0)
}

// GetCodeRequirement returns whether or not a guest code is required for a given location.
// This uses the const values defined in types.go: GuestCodeRequired, GuestCodeNotRequired, and Unknown.
// In the case that no tests have been performed, Unknown will be returned.
func GetCodeRequirement(location int64) uint {
	key := fmt.Sprintf("code_required:%d", location)
	if db.Exists(key).Val() == 0 {
		return Unknown
	}
	if db.Get(key).Val() == "true" {
		return GuestCodeRequired
	}
	return GuestCodeNotRequired
}
