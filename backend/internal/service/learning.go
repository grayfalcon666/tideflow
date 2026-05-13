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
	ListName string           `json:"list_name"`
	Total    int              `json:"total"`
	Words    []*LearningWord  `json:"words"`
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
	lists    map[uint]*models.VocabList
	wordSets map[uint]map[string]struct{}
	mu       sync.RWMutex
}

// NewLearningService creates a learning service. Call Init() after construction.
func NewLearningService(repo *repository.Repository, wbSvc *WordbankService) *LearningService {
	return &LearningService{
		repo:     repo,
		wbSvc:    wbSvc,
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
func (s *LearningService) GetLearningWords(ctx context.Context, videoID, listID uint) (*LearningWordsResponse, error) {
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

	resp := &LearningWordsResponse{
		ListName: listMeta.Name,
		Total:    len(matchWords),
		Words:    matchWords,
	}

	slog.Info("learning: 词表交集计算完成",
		"video_id", videoID,
		"list_name", listMeta.Name,
		"wordbank_size", len(rawWords),
		"vocab_size", len(wordSet),
		"matched", len(matchWords),
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

// ImportVocabLists scans the given directory for .txt files and imports them into MySQL,
// then reloads the in-memory cache.
func (s *LearningService) ImportVocabLists(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txt" {
			continue
		}

		slug := strings.TrimSuffix(entry.Name(), ".txt")
		name := slugToName(slug)

		filePath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			slog.Warn("learning: failed to read vocab file", "path", filePath, "err", err)
			continue
		}

		lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
		var wordModels []*models.VocabWord
		for _, line := range lines {
			word := strings.TrimSpace(strings.ToLower(line))
			if word == "" {
				continue
			}
			wordModels = append(wordModels, &models.VocabWord{Word: word})
		}

		if len(wordModels) == 0 {
			continue
		}

		// Upsert list
		list := &models.VocabList{Name: name, Slug: slug, Language: "en", Total: len(wordModels)}
		if err := s.repo.UpsertVocabList(ctx, list); err != nil {
			slog.Warn("learning: failed to upsert list", "slug", slug, "err", err)
			continue
		}

		// Get list ID
		allLists, _ := s.repo.GetAllVocabLists(ctx)
		var listID uint
		for _, l := range allLists {
			if l.Slug == slug {
				listID = l.ID
				break
			}
		}

		// Replace words
		if listID > 0 {
			if err := s.repo.DeleteVocabWordsByListID(ctx, listID); err != nil {
				slog.Warn("learning: failed to delete old words", "slug", slug, "err", err)
				continue
			}
			for i := range wordModels {
				wordModels[i].ListID = listID
			}
			if err := s.repo.BatchUpsertVocabWords(ctx, wordModels); err != nil {
				slog.Warn("learning: failed to insert words", "slug", slug, "err", err)
				continue
			}
		}

		slog.Info("learning: imported vocab list", "slug", slug, "count", len(wordModels))
	}

	return s.Reload(ctx)
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
