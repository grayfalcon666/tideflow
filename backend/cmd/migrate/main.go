package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tideflow/internal/config"
	"tideflow/internal/models"
)

func main() {
	cfg, err := config.Load("../.env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := gorm.Open(mysql.Open(cfg.DB.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := migrate(db); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	if err := seedVocabLists(db, cfg.VocabListsDir); err != nil {
		log.Printf("warning: failed to seed vocab lists: %v", err)
	}

	log.Println("migration completed successfully")
}

func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Account{},
		&models.Video{},
		&models.Like{},
		&models.Comment{},
		&models.Social{},
		&models.Message{},
		&models.Tag{},
		&models.VideoTag{},
		&models.OutboxMsg{},
		&models.Notification{},
		&models.Note{},
		&models.VideoSubtitle{},
		&models.VideoWordbank{},
		&models.VocabList{},
		&models.VocabWord{},
		&models.UserWord{},
		&models.UserVideoProgress{},
		&models.UserDailyLearning{},
	); err != nil {
		return err
	}

	// 复合索引：加速 Timeline 回源和热门排序查询
	indexes := []string{
		"CREATE INDEX idx_videos_author_create ON videos (author_id, create_time)",
		"CREATE INDEX idx_videos_likes_count_id ON videos (likes_count, id)",
		"CREATE INDEX idx_videos_popularity_time_id ON videos (popularity, create_time)",
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			// 忽略 "duplicate index name" 错误
			if !isDuplicateIndexErr(err) {
				return err
			}
		}
	}
	return nil
}

func isDuplicateIndexErr(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate key name") || strings.Contains(err.Error(), "index already exists"))
}

// seedVocabLists scans the given directory for .txt files, imports new ones,
// and deletes lists whose files have been removed from disk.
// File name without extension becomes the slug (e.g. "cet4.txt" → slug="cet4").
func seedVocabLists(db *gorm.DB, dir string) error {
	// 1. Scan directory for .txt files → fileSlugs set
	fileSlugs := make(map[string]string) // slug → filePath
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("vocab lists directory not found (%s), skipping seed", dir)
			return nil
		}
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
	var existingLists []models.VocabList
	db.Find(&existingLists)
	existingSlugs := make(map[string]uint) // slug → listID
	for _, l := range existingLists {
		existingSlugs[l.Slug] = l.ID
	}

	// 3. Delete lists whose files no longer exist on disk
	for slug, listID := range existingSlugs {
		if _, ok := fileSlugs[slug]; ok {
			continue
		}
		log.Printf("file removed, deleting vocab list: %s (id=%d)", slug, listID)
		db.Where("list_id = ?", listID).Delete(&models.VocabWord{})
		db.Delete(&models.VocabList{}, listID)
	}

	// 4. Import new files (skip if slug already exists in DB)
	for slug, filePath := range fileSlugs {
		if _, exists := existingSlugs[slug]; exists {
			log.Printf("vocab list already exists, skipping: %s", slug)
			continue
		}

		name := slugToName(slug)
		words, err := readWordFile(filePath)
		if err != nil {
			log.Printf("warning: failed to read %s: %v", filePath, err)
			continue
		}
		if len(words) == 0 {
			continue
		}

		// Upsert vocab list
		if err := db.Exec(`INSERT INTO vocab_lists (name, slug, language, total)
			VALUES (?, ?, 'en', ?)
			ON DUPLICATE KEY UPDATE total = VALUES(total)`,
			name, slug, len(words)).Error; err != nil {
			log.Printf("warning: failed to upsert vocab list %s: %v", slug, err)
			continue
		}

		// Get list ID
		var listID uint
		if err := db.Raw("SELECT id FROM vocab_lists WHERE slug = ?", slug).Scan(&listID).Error; err != nil || listID == 0 {
			log.Printf("warning: failed to get list ID for %s: %v", slug, err)
			continue
		}

		// Insert words
		batch := make([]models.VocabWord, 0, 500)
		for _, word := range words {
			word = strings.TrimSpace(strings.ToLower(word))
			if word == "" {
				continue
			}
			batch = append(batch, models.VocabWord{ListID: listID, Word: word})
			if len(batch) >= 500 {
				if err := db.CreateInBatches(batch, 500).Error; err != nil {
					log.Printf("warning: failed to insert words for %s: %v", slug, err)
					break
				}
				batch = batch[:0]
			}
		}
		if len(batch) > 0 {
			if err := db.CreateInBatches(batch, 500).Error; err != nil {
				log.Printf("warning: failed to insert remaining words for %s: %v", slug, err)
			}
		}

		log.Printf("seeded vocab list: %s (%d words)", slug, len(words))
	}

	return nil
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

func readWordFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var words []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			words = append(words, line)
		}
	}
	return words, scanner.Err()
}