package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	infraredis "tideflow/infra/redis"
	"tideflow/internal/models"
	"tideflow/internal/repository"
)

var (
	ErrVocabNotInit         = errors.New("vocab lists not initialized")
	ErrListNotFound         = errors.New("vocab list not found")
	ErrWordbankNotReady     = errors.New("wordbank not ready")
	ErrWordNotInWb          = errors.New("word not found in wordbank")
	ErrNoIntersection       = errors.New("no words match the selected vocab list")
	ErrBatchConflict        = errors.New("active batch belongs to a different video")
	ErrDailyQuotaExceeded   = errors.New("daily learning quota reached")
	ErrNoActiveBatch        = errors.New("no active learning batch")
)

// CommitReq is the request body for POST /learn/commit (single word).
type CommitReq struct {
	Word   string `json:"word"`
	Result string `json:"result"` // "correct" or "wrong"
}

// CommitResp is the response for POST /learn/commit.
type CommitResp struct {
	NewStatus       int `json:"new_status"`
	DailyWordsToday int `json:"daily_words_today"`
	BatchRemaining  int `json:"batch_remaining"`
}

// LearnWordsResp is the response for GET /learn/words.
type LearnWordsResp struct {
	Words          []*LearningWord `json:"words"`
	BatchTotal     int             `json:"batch_total"`
	BatchRemaining int             `json:"batch_remaining"`
	DailyRemaining *int            `json:"daily_remaining"` // nil for type mode
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
	Value             string `json:"value"`
	Usphone           string `json:"usphone"`
	Ukphone           string `json:"ukphone"`
	Definition        string `json:"definition"`
	Translation       string `json:"translation"`
	Pos               string `json:"pos"`
	FirstCaptionStart string `json:"first_caption_start"`
}

// WordCaptionsResponse is the response for GET /learn/word/:word/captions.
type WordCaptionsResponse struct {
	Word     string          `json:"word"`
	Captions []*CaptionEntry `json:"captions"`
}

// CaptionEntry represents a subtitle caption segment.
type CaptionEntry struct {
	Start   string `json:"start"`
	End     string `json:"end"`
	Content string `json:"content"`
}

type wordEntry struct {
	Value       string          `json:"value"`
	Usphone     string          `json:"usphone"`
	Ukphone     string          `json:"ukphone"`
	Definition  string          `json:"definition"`
	Translation string          `json:"translation"`
	Pos         string          `json:"pos"`
	Captions    []*CaptionEntry `json:"captions"`
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
	lists, err := s.repo.GetAllVocabLists(ctx)
	if err != nil {
		return err
	}

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
	for _, l := range s.lists {
		result = append(result, l)
	}
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].ID < result[i].ID {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

// GetLearningWords returns the current learning batch for a user.
// If a batch already exists and matches the video, returns queued words.
// Otherwise builds a new batch with sorted candidates and pushes to Redis.
func (s *LearningService) GetLearningWords(ctx context.Context, videoID, listID, accountID uint, chunkSize int, mode string) (*LearnWordsResp, error) {
	s.mu.RLock()
	wordSet, listOK := s.wordSets[listID]
	_, metaOK := s.lists[listID]
	s.mu.RUnlock()

	if !listOK || !metaOK {
		return nil, ErrListNotFound
	}

	queueKey := infraredis.UserQueue(accountID)
	batchKey := infraredis.UserBatch(accountID)

	// 1. Check existing batch
	batchInfo, err := s.cache.GetBatchInfo(ctx, batchKey)
	if err != nil {
		return nil, err
	}

	if batchInfo != nil {
		batchVID, _ := strconv.ParseUint(batchInfo["video_id"], 10, 64)
		if uint(batchVID) != videoID {
			return nil, ErrBatchConflict
		}

		// Existing batch matches video — return queued words
		queueWords, err := s.cache.GetQueueWords(ctx, queueKey)
		if err != nil {
			return nil, err
		}

		if len(queueWords) > 0 {
			words := s.hydrateWords(ctx, videoID, queueWords)
			batchTotal, _ := strconv.Atoi(batchInfo["total"])
			dailyRemaining := s.getDailyRemaining(ctx, accountID, mode)

			return &LearnWordsResp{
				Words:          words,
				BatchTotal:     batchTotal,
				BatchRemaining: len(queueWords),
				DailyRemaining: dailyRemaining,
			}, nil
		}

		// Queue is empty — batch completed, delete and fall through to build new
		s.cache.DeleteBatchAndQueue(ctx, batchKey, queueKey)
	}

	// 2. Build new batch
	return s.buildNewBatch(ctx, videoID, listID, accountID, chunkSize, mode, wordSet)
}

// buildNewBatch constructs a new learning batch: wordbank intersection, lazy evaluation, sorted queue.
func (s *LearningService) buildNewBatch(ctx context.Context, videoID, listID, accountID uint, chunkSize int, mode string, wordSet map[string]struct{}) (*LearnWordsResp, error) {
	queueKey := infraredis.UserQueue(accountID)
	batchKey := infraredis.UserBatch(accountID)

	// Get wordbank
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

	// Build wordbank map and intersection list
	wbMap := make(map[string]*LearningWord)
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
		lw := &LearningWord{
			Value:             entry.Value,
			Usphone:           entry.Usphone,
			Ukphone:           entry.Ukphone,
			Definition:        entry.Definition,
			Translation:       entry.Translation,
			Pos:               entry.Pos,
			FirstCaptionStart: firstStart,
		}
		wbMap[value] = lw
		matchWords = append(matchWords, lw)
	}

	if len(matchWords) == 0 {
		return &LearnWordsResp{Words: nil, BatchTotal: 0, BatchRemaining: 0, DailyRemaining: s.getDailyRemaining(ctx, accountID, mode)}, nil
	}

	// Get user word statuses (all, including status=0)
	wordList := make([]string, len(matchWords))
	for i, w := range matchWords {
		wordList[i] = strings.ToLower(w.Value)
	}
	userWords, err := s.repo.GetUserWordStatusesAll(ctx, accountID, wordList)
	if err != nil {
		slog.Warn("learning: failed to get user word statuses", "err", err)
	}

	// Categorize words
	now := time.Now()
	knownWords := make(map[string]repository.UserWordStatus)
	for _, uw := range userWords {
		knownWords[uw.Word] = uw
	}

	var wrongWords, reviewWords, newWords []*LearningWord
	for _, w := range matchWords {
		wordLower := strings.ToLower(w.Value)
		uw, exists := knownWords[wordLower]
		if !exists {
			newWords = append(newWords, w)
			continue
		}
		if uw.Status == 0 {
			wrongWords = append(wrongWords, w)
			continue
		}
		if now.Sub(uw.UpdatedAt) >= reviewInterval(uw.Status) {
			reviewWords = append(reviewWords, w)
		}
	}

	// Sort: wrong words by updated_at ASC, review words by updated_at ASC
	sortWordsByUpdatedAt(wrongWords, knownWords)
	sortWordsByUpdatedAt(reviewWords, knownWords)

	// Merge: wrong → review → new
	candidates := make([]*LearningWord, 0, len(wrongWords)+len(reviewWords)+len(newWords))
	candidates = append(candidates, wrongWords...)
	candidates = append(candidates, reviewWords...)
	candidates = append(candidates, newWords...)

	if len(candidates) == 0 {
		return &LearnWordsResp{Words: nil, BatchTotal: 0, BatchRemaining: 0, DailyRemaining: s.getDailyRemaining(ctx, accountID, mode)}, nil
	}

	// Check daily quota
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	remaining := s.getDailyRemainingInt(ctx, accountID, today)
	if remaining <= 0 && len(wrongWords) == 0 {
		return &LearnWordsResp{Words: nil, BatchTotal: 0, BatchRemaining: 0, DailyRemaining: intPtr(0)}, nil
	}

	// Slice to chunk_size
	if chunkSize > 0 && len(candidates) > chunkSize {
		candidates = candidates[:chunkSize]
	}

	// Enqueue to Redis
	queueStrings := make([]string, len(candidates))
	for i, w := range candidates {
		queueStrings[i] = strings.ToLower(w.Value)
	}

	batchID := fmt.Sprintf("%d:%d", videoID, now.UnixMilli())
	s.cache.PushQueueTail(ctx, queueKey, queueStrings)
	s.cache.SetBatchInfo(ctx, batchKey, map[string]interface{}{
		"video_id":    videoID,
		"list_id":     listID,
		"mode":        mode,
		"total":       len(candidates),
		"batch_id":    batchID,
		"created_at":  now.Unix(),
	})

	dailyRemaining := s.getDailyRemaining(ctx, accountID, mode)

	slog.Info("learning: new batch built",
		"account_id", accountID,
		"video_id", videoID,
		"batch_id", batchID,
		"total", len(candidates),
		"wrong", len(wrongWords),
		"review", len(reviewWords),
		"new", len(newWords),
	)

	return &LearnWordsResp{
		Words:          candidates,
		BatchTotal:     len(candidates),
		BatchRemaining: len(candidates),
		DailyRemaining: dailyRemaining,
	}, nil
}

// CommitLearning processes a single word submission (spell mode).
// Correct: status + 1, remove from queue. Wrong: status - 1, push to queue head.
func (s *LearningService) CommitLearning(ctx context.Context, accountID uint, req CommitReq) (*CommitResp, error) {
	queueKey := infraredis.UserQueue(accountID)
	batchKey := infraredis.UserBatch(accountID)
	word := strings.ToLower(req.Word)

	// 1. Validate batch exists
	batchInfo, err := s.cache.GetBatchInfo(ctx, batchKey)
	if err != nil {
		return nil, err
	}
	if batchInfo == nil {
		return nil, ErrNoActiveBatch
	}

	videoID, _ := strconv.ParseUint(batchInfo["video_id"], 10, 64)

	// 2. Remove word from queue (LREM removes one instance)
	removed := s.cache.RemoveQueueWord(ctx, queueKey, word)

	// 3. Process based on result
	db := s.repo.DB()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var newStatus int8
	var dailyDelta int

	if req.Result == "correct" {
		// Star upgrade: LEAST(status + 1, 4)
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var result struct {
				OldStatus int8
				NewStatus int8
			}
			return tx.Raw(
				`INSERT INTO user_words (account_id, word, status, updated_at) VALUES (?, ?, 1, NOW())
				 ON DUPLICATE KEY UPDATE
				   status = LEAST(COALESCE(status, 0) + 1, 4),
				   updated_at = NOW()`,
				accountID, word,
			).Scan(&result).Error
		})
		if err != nil {
			// Rollback: push word back to queue tail
			if removed > 0 {
				s.cache.PushQueueTail(ctx, queueKey, []string{word})
			}
			return nil, err
		}

		// Read back the new status
		var uw repository.UserWordStatus
		db.WithContext(ctx).Model(&models.UserWord{}).
			Select("status").
			Where("account_id = ? AND word = ?", accountID, word).
			First(&uw)
		newStatus = uw.Status

		// Count as forward improvement
		dailyDelta = 1
	} else {
		// Star downgrade: GREATEST(status - 1, 0)
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return tx.Exec(
				`INSERT INTO user_words (account_id, word, status, updated_at) VALUES (?, ?, 0, NOW())
				 ON DUPLICATE KEY UPDATE
				   status = GREATEST(COALESCE(status, 0) - 1, 0),
				   updated_at = NOW()`,
				accountID, word,
			).Error
		})
		if err != nil {
			if removed > 0 {
				s.cache.PushQueueTail(ctx, queueKey, []string{word})
			}
			return nil, err
		}

		newStatus = 0

		// Push word to queue head for immediate retry
		s.cache.PushQueueHead(ctx, queueKey, word)
		dailyDelta = 0 // wrong answers don't count toward daily goal
	}

	// 4. Update daily learning stats (only for forward improvements)
	if dailyDelta > 0 {
		_ = s.repo.UpsertUserDailyLearning(ctx, accountID, today, dailyDelta, 0)
	}

	// 5. Post-transaction: habit bitmap + cache invalidation
	habitKey := infraredis.UserHabit(now.Year(), accountID)
	s.cache.SetHabitBitmap(ctx, habitKey, now.YearDay())
	s.cache.InvalidateTodayWords(ctx, infraredis.UserTodayWords(accountID))

	// 6. Get batch remaining
	remaining, _ := s.cache.GetQueueLength(ctx, queueKey)

	// 7. Update daily today count
	dailyToday := s.getDailyToday(ctx, accountID, today)

	slog.Info("learning: commit word",
		"account_id", accountID,
		"video_id", videoID,
		"word", word,
		"result", req.Result,
		"new_status", newStatus,
		"batch_remaining", remaining,
	)

	return &CommitResp{
		NewStatus:       int(newStatus),
		DailyWordsToday: dailyToday,
		BatchRemaining:  int(remaining),
	}, nil
}

// AbortBatch deletes the user's active learning batch and queue.
func (s *LearningService) AbortBatch(ctx context.Context, accountID uint) error {
	queueKey := infraredis.UserQueue(accountID)
	batchKey := infraredis.UserBatch(accountID)
	s.cache.DeleteBatchAndQueue(ctx, batchKey, queueKey)

	slog.Info("learning: batch aborted", "account_id", accountID)
	return nil
}

// GetHabitStats returns learning habit statistics for a user in a given year.
func (s *LearningService) GetHabitStats(ctx context.Context, accountID uint, year int) (*HabitStatsResp, error) {
	if year == 0 {
		year = time.Now().Year()
	}

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

	habitKey := infraredis.UserHabit(year, accountID)
	totalDays, currentStreak := s.cache.GetHabitStats(ctx, habitKey)

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

	if cached, err := s.cache.GetTodayWords(ctx, key); err == nil && cached != "" {
		var items []TodayWordItem
		if err := json.Unmarshal([]byte(cached), &items); err == nil {
			return items, nil
		}
	}

	records, err := s.repo.GetUserWordsToday(ctx, accountID)
	if err != nil {
		return nil, err
	}

	items := make([]TodayWordItem, 0, len(records))
	for _, r := range records {
		items = append(items, TodayWordItem{Word: r.Word, Status: r.Status})
	}

	if data, err := json.Marshal(items); err == nil {
		s.cache.SetTodayWords(ctx, key, string(data))
	}

	return items, nil
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
			return &WordCaptionsResponse{
				Word:     entry.Value,
				Captions: entry.Captions,
			}, nil
		}
	}

	return nil, ErrWordNotInWb
}

// ImportVocabLists scans the given directory for .txt files and imports new ones into MySQL.
func (s *LearningService) ImportVocabLists(ctx context.Context, dir string) error {
	fileSlugs := make(map[string]string)
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

	existingLists, err := s.repo.GetAllVocabLists(ctx)
	if err != nil {
		return err
	}
	existingSlugs := make(map[string]*models.VocabList)
	for _, l := range existingLists {
		existingSlugs[l.Slug] = l
	}

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

	for slug, filePath := range fileSlugs {
		if _, exists := existingSlugs[slug]; exists {
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

		allLists, _ := s.repo.GetAllVocabLists(ctx)
		var listID uint
		for _, l := range allLists {
			if l.Slug == slug {
				listID = l.ID
				break
			}
		}
		if listID == 0 {
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

// Reload reloads all vocab lists and words from MySQL into memory.
func (s *LearningService) Reload(ctx context.Context) error {
	return s.Init(ctx)
}

// =================================================================
// Internal helpers
// =================================================================

// hydrateWords maps word strings back to full LearningWord objects via wordbank.
func (s *LearningService) hydrateWords(ctx context.Context, videoID uint, queueWords []string) []*LearningWord {
	wbResp, err := s.wbSvc.GetWordbank(ctx, videoID)
	if err != nil {
		// Wordbank not available — return minimal objects
		words := make([]*LearningWord, len(queueWords))
		for i, w := range queueWords {
			words[i] = &LearningWord{Value: w}
		}
		return words
	}

	var rawWords []json.RawMessage
	if err := json.Unmarshal(wbResp.Words, &rawWords); err != nil {
		words := make([]*LearningWord, len(queueWords))
		for i, w := range queueWords {
			words[i] = &LearningWord{Value: w}
		}
		return words
	}

	// Build lookup map from wordbank
	wbMap := make(map[string]*LearningWord, len(rawWords))
	for _, raw := range rawWords {
		var entry wordEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		value := strings.ToLower(entry.Value)
		firstStart := ""
		if len(entry.Captions) > 0 {
			firstStart = entry.Captions[0].Start
		}
		wbMap[value] = &LearningWord{
			Value:             entry.Value,
			Usphone:           entry.Usphone,
			Ukphone:           entry.Ukphone,
			Definition:        entry.Definition,
			Translation:       entry.Translation,
			Pos:               entry.Pos,
			FirstCaptionStart: firstStart,
		}
	}

	words := make([]*LearningWord, len(queueWords))
	for i, w := range queueWords {
		wLower := strings.ToLower(w)
		if lw, ok := wbMap[wLower]; ok {
			words[i] = lw
		} else {
			words[i] = &LearningWord{Value: w}
		}
	}
	return words
}

// getDailyRemaining returns the remaining daily quota as *int (nil for type mode).
func (s *LearningService) getDailyRemaining(ctx context.Context, accountID uint, mode string) *int {
	if mode != "spell" {
		return nil
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	v := s.getDailyRemainingInt(ctx, accountID, today)
	return &v
}

// getDailyRemainingInt returns the remaining daily quota as int.
func (s *LearningService) getDailyRemainingInt(ctx context.Context, accountID uint, today time.Time) int {
	account, err := s.repo.GetAccountByID(ctx, accountID)
	if err != nil || account == nil {
		return 0
	}
	if account.DailyGoal <= 0 {
		return 999999 // no limit
	}
	todayWords := s.getDailyToday(ctx, accountID, today)
	remaining := account.DailyGoal - todayWords
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// getDailyToday returns today's learned word count.
func (s *LearningService) getDailyToday(ctx context.Context, accountID uint, today time.Time) int {
	record, err := s.repo.GetUserDailyLearningByDate(ctx, accountID, today)
	if err != nil || record == nil {
		return 0
	}
	return record.WordsCount
}

// sortWordsByUpdatedAt sorts words by their user_words.updated_at ASC.
func sortWordsByUpdatedAt(words []*LearningWord, known map[string]repository.UserWordStatus) {
	for i := 0; i < len(words); i++ {
		for j := i + 1; j < len(words); j++ {
			wi := known[strings.ToLower(words[i].Value)]
			wj := known[strings.ToLower(words[j].Value)]
			if wj.UpdatedAt.Before(wi.UpdatedAt) {
				words[i], words[j] = words[j], words[i]
			}
		}
	}
}

func intPtr(v int) *int { return &v }

func parseVocabFile(filePath, slug string) ([]*models.VocabWord, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	var wordModels []*models.VocabWord
	for _, line := range lines {
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
		wordModels = append(wordModels, &models.VocabWord{Word: word})
	}
	return wordModels, nil
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
