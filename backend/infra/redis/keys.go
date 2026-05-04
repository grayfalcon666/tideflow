package redis

import "fmt"

const (
	Version = "v1"

	KeyAccount         = "account:%d"
	KeyAccountRefresh  = "account:%d:refresh"
	KeyRefreshToken    = "refresh:%s"
	KeyVideoEntity     = "video:entity:%d"
	KeyVideoDetail     = "video:detail:%d"
	KeyFeedGlobal      = "feed:global"
	KeyHotVideo        = "hot:video:%s:%s"
	KeyHotMerge        = "hot:merge:%s:%s"
	KeyOutbox          = "outbox:%d"
	KeyInbox           = "inbox:%d"
	KeyFeedCache       = "feed:followcache:%d:before:%s:limit:%d"
	KeySFLabel         = "sf:%s"
	KeyRateLimit       = "ratelimit:%s:%s"
	KeyLock            = "lock:%s"
	KeyBigVMark        = "bigv:mark:%d"
)

func AccountToken(uid uint) string {
	return fmt.Sprintf("%s:"+KeyAccount, Version, uid)
}

func AccountRefreshToken(uid uint) string {
	return fmt.Sprintf("%s:"+KeyAccountRefresh, Version, uid)
}

func RefreshToUID(token string) string {
	return fmt.Sprintf("%s:"+KeyRefreshToken, Version, token)
}

func VideoEntity(id uint) string {
	return fmt.Sprintf("%s:"+KeyVideoEntity, Version, id)
}

func VideoDetail(id uint) string {
	return fmt.Sprintf("%s:"+KeyVideoDetail, Version, id)
}

func FeedGlobal() string {
	return fmt.Sprintf("%s:"+KeyFeedGlobal, Version)
}

func HotVideo(window, ts string) string {
	return fmt.Sprintf("%s:"+KeyHotVideo, Version, window, ts)
}

func HotMerge(window, ts string) string {
	return fmt.Sprintf("%s:"+KeyHotMerge, Version, window, ts)
}

func Outbox(uid uint) string {
	return fmt.Sprintf("%s:"+KeyOutbox, Version, uid)
}

func Inbox(uid uint) string {
	return fmt.Sprintf("%s:"+KeyInbox, Version, uid)
}

func FeedCache(uid uint, ts string, limit int) string {
	return fmt.Sprintf("%s:"+KeyFeedCache, Version, uid, ts, limit)
}

func SFLabel(name string) string {
	return fmt.Sprintf("%s:"+KeySFLabel, Version, name)
}

func RateLimit(action, subject string) string {
	return fmt.Sprintf("%s:"+KeyRateLimit, Version, action, subject)
}

// LockKey returns a versioned lock key. Prefer typed functions below.
func LockKey(target string) string {
	return fmt.Sprintf("%s:"+KeyLock, Version, target)
}

// LockDetail returns the distributed lock key for video detail cache.
func LockDetail(id uint) string {
	return fmt.Sprintf("%s:lock:detail:%d", Version, id)
}

func BigVMark(uid uint) string {
	return fmt.Sprintf("%s:"+KeyBigVMark, Version, uid)
}