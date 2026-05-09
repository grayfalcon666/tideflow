package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"tideflow/internal/models"
	"tideflow/internal/repository"
)

var (
	ErrNoteNotFound      = errors.New("note not found")
	ErrNotNoteAuthor     = errors.New("not the author of this note")
	ErrNoteLimitExceeded = errors.New("note limit exceeded for this video")
)

const maxNotesPerVideo = 500
const maxNoteContentLen = 2000

type NoteService struct {
	repo *repository.Repository
}

func NewNoteService(repo *repository.Repository) *NoteService {
	return &NoteService{repo: repo}
}

func (s *NoteService) CreateNote(ctx context.Context, videoID, authorID uint, username string, timestamp float64, content string) (*models.Note, error) {
	if content == "" || len([]rune(content)) > maxNoteContentLen {
		return nil, fmt.Errorf("content is required and must be ≤ %d characters", maxNoteContentLen)
	}

	video, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}

	if timestamp < 0 || timestamp > video.Duration {
		return nil, fmt.Errorf("timestamp out of range [0, %.1f]", video.Duration)
	}

	count, err := s.repo.CountNotesByVideo(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to count notes: %w", err)
	}
	if count >= maxNotesPerVideo {
		return nil, ErrNoteLimitExceeded
	}

	note := &models.Note{
		VideoID:   videoID,
		AuthorID:  authorID,
		Username:  username,
		Timestamp: timestamp,
		Content:   content,
	}
	if err := s.repo.CreateNote(ctx, note); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}
	return note, nil
}

func (s *NoteService) GetNotes(ctx context.Context, videoID uint, cursor string, limit int) ([]*models.Note, *string, bool, error) {
	var timestampCursor float64
	if cursor != "" && cursor != "0" {
		var err error
		timestampCursor, err = strconv.ParseFloat(cursor, 64)
		if err != nil {
			timestampCursor = 0
		}
	}

	notes, err := s.repo.GetNotesByVideo(ctx, videoID, timestampCursor, limit+1)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to get notes: %w", err)
	}

	hasMore := len(notes) > limit
	if hasMore {
		notes = notes[:limit]
	}

	var nextCursor *string
	if hasMore && len(notes) > 0 {
		c := fmt.Sprintf("%.1f", notes[len(notes)-1].Timestamp)
		nextCursor = &c
	}

	return notes, nextCursor, hasMore, nil
}

func (s *NoteService) DeleteNote(ctx context.Context, noteID, authorID uint) error {
	note, err := s.repo.GetNoteByID(ctx, noteID)
	if err != nil {
		return ErrNoteNotFound
	}

	if note.AuthorID != authorID {
		return ErrNotNoteAuthor
	}

	return s.repo.SoftDeleteNote(ctx, noteID)
}
