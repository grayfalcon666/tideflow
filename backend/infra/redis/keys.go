package redis

import "fmt"

const (
	Version = "v1"

	KeyAccount         = "account:%d"
	KeyAccountRefresh  = "account:%d:refresh"
	KeyRefreshToken    = "refresh:%s"
	KeyVideoEntity     = "video:entity:%d"
	KeyFeedGlobal      = "feed:global"
	KeyHotVideo        = "hot:video:%s:%s"
	KeyHotMerge        = "hot:merge:%s:%s"
	KeyOutbox          = "outbox:%d"
	KeyInbox           = "inbox:%d"
	KeyFeedCache       = "feed:followcache:%d:before:%s:limit:%d"
	KeySFLabel         = "sf:%s"
	KeyRateLimit       = "ratelimit:%s:%s"
	KeyBigVMark        = "bigv:mark:%d"
	KeyViewLimit      = "ratelimit:view:%d:%d"
	KeyViewCount      = "count:views:%d"
	KeyDirtyVideos    = "dirty:videos"
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

func BigVMark(uid uint) string {
	return fmt.Sprintf("%s:"+KeyBigVMark, Version, uid)
}

func ViewLimit(userID, videoID uint) string {
	return fmt.Sprintf("%s:"+KeyViewLimit, Version, userID, videoID)
}

func ViewCount(videoID uint) string {
	return fmt.Sprintf("%s:"+KeyViewCount, Version, videoID)
}

func DirtyVideos() string {
	return fmt.Sprintf("%s:"+KeyDirtyVideos, Version)
}