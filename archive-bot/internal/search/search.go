// Package search provides Persian text normalization and pg_trgm fuzzy search helpers.
package search

import (
	"context"
	"strings"
	"unicode"

	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/models"
)

// Normalize normalizes a Persian/Arabic query string for consistent matching:
//   - Arabic ي  → Persian ی
//   - Arabic ك  → Persian ک
//   - Remove diacritics (harakat U+064B–U+065F)
//   - Remove zero-width non-joiner (U+200C)
//   - Collapse extra whitespace
func Normalize(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case 'ي':
			sb.WriteRune('ی')
		case 'ك':
			sb.WriteRune('ک')
		case '\u200C': // ZWNJ — skip
		default:
			if r >= 0x064B && r <= 0x065F { // diacritics
				continue
			}
			if unicode.IsMark(r) {
				continue
			}
			sb.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

// Search performs a pg_trgm similarity search across title, tags, and description.
// Results are ordered by similarity score (best match first).
//
// Requires the pg_trgm extension and the GIN index created by db.Migrate().
// To swap to a different search backend: replace this function with one that
// queries Elasticsearch, Meilisearch, etc. — the handler only calls this function.
func Search(ctx context.Context, db *gorm.DB, query string, limit int) ([]models.File, error) {
	normalized := Normalize(strings.TrimSpace(query))
	if normalized == "" {
		return nil, nil
	}

	var files []models.File
	col := "title || ' ' || tags || ' ' || description"
	err := db.WithContext(ctx).
		Preload("Category").
		Where("similarity("+col+", ?) > 0.1", normalized).
		Order(gorm.Expr("similarity("+col+", ?) DESC", normalized)).
		Limit(limit).
		Find(&files).Error

	return files, err
}

// RelatedFiles returns files from the same category as the given file,
// falling back to files sharing any tag.
func RelatedFiles(ctx context.Context, db *gorm.DB, file models.File, limit int) ([]models.File, error) {
	var files []models.File

	q := db.WithContext(ctx).
		Where("id != ?", file.ID).
		Limit(limit)

	if file.CategoryID != nil {
		q = q.Where("category_id = ?", file.CategoryID)
	} else if file.Tags != "" {
		// Match on any shared tag
		firstTag := strings.Split(file.Tags, ",")[0]
		q = q.Where("tags LIKE ?", "%"+strings.TrimSpace(firstTag)+"%")
	}

	return files, q.Find(&files).Error
}
