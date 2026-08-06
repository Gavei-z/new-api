package model

import "sync"

// User quota mutations also update a Redis mirror. Serialize DB + cache
// updates per user so an absolute refresh after a Stripe transaction cannot
// overwrite a concurrent request's quota delta.
const userQuotaMutationLockCount = 256

var userQuotaMutationLocks [userQuotaMutationLockCount]sync.Mutex

func userQuotaMutationLock(userId int) *sync.Mutex {
	index := userId % userQuotaMutationLockCount
	if index < 0 {
		index = -index
	}
	return &userQuotaMutationLocks[index]
}
