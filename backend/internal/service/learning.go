package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	infraredis "tideflow/infra/redis"
	"tideflow/internal/models"
	"tideflow/internal/repository"
)

var (
	ErrVocabNotInit     = errors.New("vocab lists not initialized")
	ErrListNotFound     = errors.New("vocab list not found")
	ErrWordbankNotReady = errors.New("wordbank not ready")
	ErrWordNotInWb      = errors.New("word not found in wordbank")
	ErrNoIntersection   = errors.New("no words match the selected vocab list")
)

// LearningWordsResponse is the response for GET /learn/words.
type LearningWordsResponse struct {
	ListName        string          `json:"list_name"`
	Total           int             `json:"total"`
	LearnedCount    int             `json:"learned_count"`
	UnmasteredTotal int             `json:"unmastered_total"`
	Words           []*LearningWord `json:"words"`
}

// WordStatus represents a word practice result.
type WordStatus struct {
	Word   string `json:"word"`
	Status int8   `json:"status"`
}

// CommitLearningReq is the request body for POST /learn/commit.
type CommitLearningReq struct {
	VideoID        uint         `json:"video_id"`
	WordsPracticed []WordStatus `json:"words_practiced"`
	IsRetry        bool         `json:"is_retry"`
}

// ResetProgressReq is the request body for POST /learn/progress/reset.
type ResetProgressReq struct {
	VideoID uint `json:"video_id"`
}

// HabitStatsResp is the response for GET /learn/habit/stats.
type HabitStatsResp struct {
	Heatmap       []DailyHeatmapEntry `json:"heatmap"`
	TotalDays     int                 `json:"total_days"`
	CurrentStreak int                 `json:"current_streak"`
	TodayWords    int                 `json:"today_words"`
	TodayVideos   int                 `json:"today_videos"`
}

// DailyHeatmapEntry is a single day entry in the heatmap.
type DailyHeatmapEntry struct {
	Date       string `json:"date"`
	WordsCount int    `json:"words_count"`
}

// TodayWordItem represents a word practiced today.
type TodayWordItem struct {
	Word   string `json:"word"`
	Status int8   `json:"status"`
}

// LearningWord is a single word entry in the learning list.
type LearningWord struct {
	Value              string `json:"value"`
	Usphone            string `json:"usphone"`
	Ukphone            string `json:"ukphone"`
	Definition         string `json:"definition"`
	Translation        string `json:"translation"`
	Pos                string `json:"pos"`
	FirstCaptionStart  string `json:"first_caption_start"`
}

// WordCaptionsResponse is the response for GET /learn/word/:word/captions.
type WordCaptionsResponse struct {
	Word     string            `json:"word"`
	Captions []*CaptionEntry   `json:"captions"`
}

// CaptionEntry represents a subtitle caption segment.
type CaptionEntry struct {
	Start   string `json:"start"`
	End     string `json:"end"`
	Content string `json:"content"`
}

type wordEntry struct {
	Value    string          `json:"value"`
	Usphone  string          `json:"usphone"`
	Ukphone  string          `json:"ukphone"`
	Definition string        `json:"definition"`
	Translation string       `json:"translation"`
	Pos      string          `json:"pos"`
	Captions []*CaptionEntry `json:"captions"`
}

// LearningService provides vocabulary-based learning list computation.
// Vocab lists are loaded into process memory on Init() for microsecond-level intersection.
type LearningService struct {
	repo     *repository.Repository
	wbSvc    *WordbankService
	cache    *infraredis.Cache
	lists    map[uint]*models.VocabList
	wordSets map[uint]map[string]struct{}
	mu       sync.RWMutex
}

// NewLearningService creates a learning service. Call Init() after construction.
func NewLearningService(repo *repository.Repository, wbSvc *WordbankService, cache *infraredis.Cache) *LearningService {
	return &LearningService{
		repo:     repo,
		wbSvc:    wbSvc,
		cache:    cache,
		lists:    make(map[uint]*models.VocabList),
		wordSets: make(map[uint]map[string]struct{}),
	}
}

// Init loads all vocab lists and words from MySQL into memory.
// Fails fast if MySQL is unavailable.
func (s *LearningService) Init(ctx context.Context) error {
	// Load lists
	lists, err := s.repo.GetAllVocabLists(ctx)
	if err != nil {
		return err
	}

	// Load all words
	words, err := s.repo.GetAllVocabWords(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lists = make(map[uint]*models.VocabList, len(lists))
	for _, l := range lists {
		s.lists[l.ID] = l
	}

	s.wordSets = make(map[uint]map[string]struct{})
	for _, w := range words {
		if _, ok := s.wordSets[w.ListID]; !ok {
			s.wordSets[w.ListID] = make(map[string]struct{})
		}
		s.wordSets[w.ListID][w.Word] = struct{}{}
	}

	slog.Info("learning: vocab loaded", "lists", len(s.lists), "words", len(words))

	if len(s.lists) == 0 {
		return ErrVocabNotInit
	}
	return nil
}

// GetLists returns all vocab lists from memory, sorted by ID.
func (s *LearningService) GetLists() ([]*models.VocabList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.lists) == 0 {
		return nil, ErrVocabNotInit
	}

	result := make([]*models.VocabList, 0, len(s.lists))
	// Return lists ordered by ID
	for _, l := range s.lists {
		result = append(result, l)
	}
	// Sort by ID (simple insertion since maps are small)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].ID < result[i].ID {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

// GetLearningWords computes the intersection of a video's wordbank with a vocab list.
// If accountID > 0, filters out mastered words and applies chunk slicing based on video progress.
// chunkSize <= 0 means return all words without slicing.
func (s *LearningService) GetLearningWords(ctx context.Context, videoID, listID, accountID uint, chunkSize int) (*LearningWordsResponse, error) {
	s.mu.RLock()
	wordSet, listOK := s.wordSets[listID]
	listMeta, metaOK := s.lists[listID]
	s.mu.RUnlock()

	if !listOK || !metaOK {
		return nil, ErrListNotFound
	}

	// Get wordbank (reuses existing WordbankService with L2/L3 cache chain)
	wbResp, err := s.wbSvc.GetWordbank(ctx, videoID)
	if err != nil {
		if errors.Is(err, ErrWordbankFailed) || errors.Is(err, ErrWordbankNotFound) {
			return nil, ErrWordbankNotReady
		}
		return nil, err
	}

	// Unmarshal words
	var rawWords []json.RawMessage
	if err := json.Unmarshal(wbResp.Words, &rawWords); err != nil {
		return nil, err
	}

	// Intersection: check each word's value against the vocab set
	var matchWords []*LearningWord
	for _, raw := range rawWords {
		var entry wordEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}

		value := strings.ToLower(entry.Value)
		if _, ok := wordSet[value]; !ok {
			continue
		}

		firstStart := ""
		if len(entry.Captions) > 0 {
			firstStart = entry.Captions[0].Start
		}

		matchWords = append(matchWords, &LearningWord{
			Value:             entry.Value,
			Usphone:            entry.Usphone,
			Ukphone:            entry.Ukphone,
			Definition:         entry.Definition,
			Translation:        entry.Translation,
			Pos:                entry.Pos,
			FirstCaptionStart:  firstStart,
		})
	}

	// User-specific filtering and chunk slicing
	var learnedCount int
	unmastered := matchWords
	if accountID > 0 {
		// Load progress cursor
		progress, err := s.repo.GetUserVideoProgress(ctx, accountID, videoID)
		if err != nil {
			slog.Warn("learning: failed to get video progress", "err", err)
		}
		if progress != nil {
			learnedCount = progress.LearnedCount
		}

		// Filter: exclude words that are still within their review interval
		wordList := make([]string, len(unmastered))
		for i, w := range unmastered {
			wordList[i] = strings.ToLower(w.Value)
		}
		userWords, err := s.repo.GetUserWordStatuses(ctx, accountID, wordList)
		if err != nil {
			slog.Warn("learning: failed to get user word statuses", "err", err)
		} else {
			inWindow := make(map[string]struct{})
			for _, uw := range userWords {
				if inReviewWindow(uw.Status, uw.UpdatedAt) {
					inWindow[uw.Word] = struct{}{}
				}
			}
			var filtered []*LearningWord
			for _, w := range unmastered {
				if _, skip := inWindow[strings.ToLower(w.Value)]; !skip {
					filtered = append(filtered, w)
				}
			}
			unmastered = filtered
		}

		// Cursor overflow defense
		if learnedCount >= len(unmastered) {
			return &LearningWordsResponse{
				ListName:        listMeta.Name,
				Total:           len(matchWords),
				LearnedCount:    learnedCount,
				UnmasteredTotal: len(unmastered),
				Words:           nil,
			}, nil
		}

		// Dynamic chunk slicing
		if chunkSize > 0 {
			end := learnedCount + chunkSize
			if end > len(unmastered) {
				end = len(unmastered)
			}
			unmastered = unmastered[learnedCount:end]
		}
	}

	resp := &LearningWordsResponse{
		ListName:        listMeta.Name,
		Total:           len(matchWords),
		LearnedCount:    learnedCount,
		UnmasteredTotal: len(unmastered),
		Words:           unmastered,
	}

	slog.Info("learning: 词表交集计算完成",
		"video_id", videoID,
		"list_name", listMeta.Name,
		"wordbank_size", len(rawWords),
		"vocab_size", len(wordSet),
		"matched", len(matchWords),
		"learned_count", learnedCount,
		"unmastered_total", len(unmastered),
	)
	return resp, nil
}

// GetWordCaptions returns the caption entries for a specific word from a video's wordbank.
func (s *LearningService) GetWordCaptions(ctx context.Context, videoID uint, word string) (*WordCaptionsResponse, error) {
	wbResp, err := s.wbSvc.GetWordbank(ctx, videoID)
	if err != nil {
		if errors.Is(err, ErrWordbankFailed) || errors.Is(err, ErrWordbankNotFound) {
			return nil, ErrWordbankNotReady
		}
		return nil, err
	}

	var rawWords []json.RawMessage
	if err := json.Unmarshal(wbResp.Words, &rawWords); err != nil {
		return nil, err
	}

	wordLower := strings.ToLower(word)
	for _, raw := range rawWords {
		var entry wordEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		if strings.ToLower(entry.Value) == wordLower {
			slog.Info("learning: 单词语境命中",
				"video_id", videoID,
				"word", entry.Value,
				"captions_count", len(entry.Captions),
			)
			return &WordCaptionsResponse{
				Word:     entry.Value,
				Captions: entry.Captions,
			}, nil
		}
	}

	slog.Info("learning: 单词不在词库中", "video_id", videoID, "word", word)
	return nil, ErrWordNotInWb
}

// ImportVocabLists scans the given directory for .txt files and imports new ones into MySQL.
// Lists whose corresponding file no longer exists on disk are deleted from MySQL.
// Lists whose slug already exists in the database are skipped entirely.
func (s *LearningService) ImportVocabLists(ctx context.Context, dir string) error {
	// 1. Scan directory for .txt files → fileSlugs set
	fileSlugs := make(map[string]string) // slug → filePath
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txt" {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".txt")
		fileSlugs[slug] = filepath.Join(dir, entry.Name())
	}

	// 2. Get existing lists from DB
	existingLists, err := s.repo.GetAllVocabLists(ctx)
	if err != nil {
		return err
	}
	existingSlugs := make(map[string]*models.VocabList) // slug → list
	for _, l := range existingLists {
		existingSlugs[l.Slug] = l
	}

	// 3. Delete lists whose files no longer exist on disk
	for slug, list := range existingSlugs {
		if _, ok := fileSlugs[slug]; ok {
			continue
		}
		slog.Info("learning: file removed, deleting vocab list", "slug", slug, "list_id", list.ID)
		if err := s.repo.DeleteVocabWordsByListID(ctx, list.ID); err != nil {
			slog.Warn("learning: failed to delete words for removed list", "slug", slug, "err", err)
			continue
		}
		if err := s.repo.DeleteVocabList(ctx, list.ID); err != nil {
			slog.Warn("learning: failed to delete list", "slug", slug, "err", err)
		}
	}

	// 4. Import new files (skip if slug already exists in DB)
	for slug, filePath := range fileSlugs {
		if _, exists := existingSlugs[slug]; exists {
			slog.Info("learning: list already exists, skipping", "slug", slug)
			continue
		}

		name := slugToName(strings.ToLower(slug))
		wordModels, err := parseVocabFile(filePath, slug)
		if err != nil {
			slog.Warn("learning: failed to parse vocab file", "slug", slug, "err", err)
			continue
		}
		if len(wordModels) == 0 {
			continue
		}

		list := &models.VocabList{Name: name, Slug: slug, Language: "en", Total: len(wordModels)}
		if err := s.repo.UpsertVocabList(ctx, list); err != nil {
			slog.Warn("learning: failed to upsert list", "slug", slug, "err", err)
			continue
		}

		// Re-fetch to get the assigned ID
		allLists, _ := s.repo.GetAllVocabLists(ctx)
		var listID uint
		for _, l := range allLists {
			if l.Slug == slug {
				listID = l.ID
				break
			}
		}
		if listID == 0 {
			slog.Warn("learning: failed to get list ID after upsert", "slug", slug)
			continue
		}

		for i := range wordModels {
			wordModels[i].ListID = listID
		}
		if err := s.repo.BatchUpsertVocabWords(ctx, wordModels); err != nil {
			slog.Warn("learning: failed to insert words", "slug", slug, "err", err)
			continue
		}

		slog.Info("learning: imported new vocab list", "slug", slug, "count", len(wordModels), "sample_words", sampleWords(wordModels))
	}

	return s.Reload(ctx)
}

// parseVocabFile reads a .txt file and extracts words (one per line).
// Words are parsed as everything before the first space, '[', or tab.
func parseVocabFile(filePath, slug string) ([]*models.VocabWord, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	slog.Info("learning: file read", "slug", slug, "total_lines", len(lines))
	var wordModels []*models.VocabWord
	for idx, line := range lines {
		raw := strings.TrimSpace(line)
		if raw == "" {
			continue
		}
		end := len(raw)
		for i, c := range raw {
			if c == ' ' || c == '[' || c == '\t' {
				end = i
				break
			}
		}
		word := strings.ToLower(raw[:end])
		if word == "" {
			continue
		}
		if idx < 5 {
			slog.Info("learning: parsed line", "idx", idx, "raw", raw, "word", word)
		}
		wordModels = append(wordModels, &models.VocabWord{Word: word})
	}
	return wordModels, nil
}

// CommitLearning processes a learning session submission.
// Updates video progress, user word statuses, daily learning rollup, and habit bitmap.
func (s *LearningService) CommitLearning(ctx context.Context, accountID uint, req CommitLearningReq) error {
	db := s.repo.DB()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		delta := len(req.WordsPracticed)

		// 1. Upsert video progress (skip for retry — cursor already advanced)
		if !req.IsRetry {
			if err := tx.Exec(
				`INSERT INTO user_video_progress (account_id, video_id, learned_count, updated_at)
				 VALUES (?, ?, ?, NOW())
				 ON DUPLICATE KEY UPDATE learned_count = learned_count + ?, updated_at = NOW()`,
				accountID, req.VideoID, delta, delta,
			).Error; err != nil {
				return err
			}
		}

		// 2. Batch upsert user words (always apply — star changes take effect)
		for _, w := range req.WordsPracticed {
			if err := tx.Exec(
				`INSERT INTO user_words (account_id, word, status, updated_at) VALUES (?, ?, ?, NOW())
				 ON DUPLICATE KEY UPDATE
				   status = IF(VALUES(status) = 0, 0, LEAST(status + 1, 4)),
				   updated_at = NOW()`,
				accountID, strings.ToLower(w.Word), w.Status,
			).Error; err != nil {
				return err
			}
		}

		// 3. Upsert daily learning rollup (skip for retry — don't double count)
		if !req.IsRetry {
			if err := tx.Exec(
				`INSERT INTO user_daily_learnings (account_id, date, words_count, videos_count, updated_at)
				 VALUES (?, ?, ?, 1, NOW())
				 ON DUPLICATE KEY UPDATE words_count = words_count + ?, updated_at = NOW()`,
				accountID, today, delta,
			).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 4. Set habit bitmap (after transaction commit)
	habitKey := infraredis.UserHabit(now.Year(), accountID)
	s.cache.SetHabitBitmap(ctx, habitKey, now.YearDay())

	// 5. Invalidate today's words cache
	if !req.IsRetry {
		s.cache.InvalidateTodayWords(ctx, infraredis.UserTodayWords(accountID))
	}

	slog.Info("learning: commit learning",
		"account_id", accountID,
		"video_id", req.VideoID,
		"words_count", len(req.WordsPracticed),
	)
	return nil
}

// GetHabitStats returns learning habit statistics for a user in a given year.
func (s *LearningService) GetHabitStats(ctx context.Context, accountID uint, year int) (*HabitStatsResp, error) {
	if year == 0 {
		year = time.Now().Year()
	}

	// 1. Get daily learning records from MySQL
	records, err := s.repo.GetUserDailyLearnings(ctx, accountID, year)
	if err != nil {
		return nil, err
	}

	heatmap := make([]DailyHeatmapEntry, 0, len(records))
	for _, r := range records {
		heatmap = append(heatmap, DailyHeatmapEntry{
			Date:       r.Date.Format("2006-01-02"),
			WordsCount: r.WordsCount,
		})
	}

	// 2. Get bitmap stats from Redis
	habitKey := infraredis.UserHabit(year, accountID)
	totalDays, currentStreak := s.cache.GetHabitStats(ctx, habitKey)

	// 3. Find today's stats from heatmap
	todayStr := time.Now().Format("2006-01-02")
	var todayWords, todayVideos int
	for _, r := range records {
		if r.Date.Format("2006-01-02") == todayStr {
			todayWords = r.WordsCount
			todayVideos = r.VideosCount
			break
		}
	}

	return &HabitStatsResp{
		Heatmap:       heatmap,
		TotalDays:     totalDays,
		CurrentStreak: currentStreak,
		TodayWords:    todayWords,
		TodayVideos:   todayVideos,
	}, nil
}

// GetTodayWords returns all words the user practiced today, with Redis-first cache.
func (s *LearningService) GetTodayWords(ctx context.Context, accountID uint) ([]TodayWordItem, error) {
	key := infraredis.UserTodayWords(accountID)

	// 1. Try Redis
	if cached, err := s.cache.GetTodayWords(ctx, key); err == nil && cached != "" {
		var items []TodayWordItem
		if err := json.Unmarshal([]byte(cached), &items); err == nil {
			return items, nil
		}
	}

	// 2. Fallback to MySQL
	records, err := s.repo.GetUserWordsToday(ctx, accountID)
	if err != nil {
		return nil, err
	}

	items := make([]TodayWordItem, 0, len(records))
	for _, r := range records {
		items = append(items, TodayWordItem{Word: r.Word, Status: r.Status})
	}

	// 3. Write back to Redis
	if data, err := json.Marshal(items); err == nil {
		s.cache.SetTodayWords(ctx, key, string(data))
	}

	return items, nil
}

// reviewInterval returns the no-repeat window for a given star level.
func reviewInterval(status int8) time.Duration {
	switch status {
	case 1:
		return 12 * time.Hour
	case 2:
		return 3 * 24 * time.Hour
	case 3:
		return 7 * 24 * time.Hour
	case 4:
		return 30 * 24 * time.Hour
	default:
		return 0
	}
}

// inReviewWindow returns true if the word is still within its review interval
// and should be excluded from the learning list.
func inReviewWindow(status int8, updatedAt time.Time) bool {
	if status == 0 {
		return false
	}
	return time.Since(updatedAt) < reviewInterval(status)
}

// ResetProgress resets the video learning cursor for a user.
func (s *LearningService) ResetProgress(ctx context.Context, accountID, videoID uint) error {
	return s.repo.DB().WithContext(ctx).
		Exec("DELETE FROM user_video_progress WHERE account_id = ? AND video_id = ?", accountID, videoID).
		Error
}

func sampleWords(words []*models.VocabWord) []string {
	n := 10
	if len(words) < n {
		n = len(words)
	}
	res := make([]string, n)
	for i := 0; i < n; i++ {
		res[i] = words[i].Word
	}
	return res
}

// Reload reloads all vocab lists and words from MySQL into memory.
func (s *LearningService) Reload(ctx context.Context) error {
	return s.Init(ctx)
}

func slugToName(slug string) string {
	names := map[string]string{
		"gaokao": "高考词汇",
		"cet4":   "四级词汇",
		"cet6":   "六级词汇",
		"ielts":  "雅思词汇",
		"toefl":  "托福词汇",
		"gre":    "GRE词汇",
	}
	if name, ok := names[slug]; ok {
		return name
	}
	return strings.ToUpper(slug)
}
