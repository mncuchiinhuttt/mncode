package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)
func EvolveMemory(store *HierarchicalStore, lesson ReflectiveLesson, tier MemoryTier) (*MemoryItem, bool, error) {
	if store == nil {
		return nil, false, fmt.Errorf("store is required")
	}

	topicSlug := normalizeTopic(lesson.Topic)
	if topicSlug == "" {
		topicSlug = "general-insight"
	}

	existing := store.ListAll()
	var priorMatch *MemoryItem

	for i := range existing {
		item := &existing[i]
		if strings.EqualFold(normalizeTopic(item.Topic), topicSlug) {
			priorMatch = item
			break
		}
	}

	supersedesID := ""
	hitCount := 1
	if priorMatch != nil {
		supersedesID = priorMatch.ID
		hitCount = priorMatch.HitCount + 1
		// Delete old version before saving evolved version
		_ = store.Delete(priorMatch.ID)
	}

	hash := sha256.Sum256([]byte(topicSlug + lesson.Summary + fmt.Sprintf("-%d", time.Now().UnixNano())))
	newItem := MemoryItem{
		ID:           "mem-" + hex.EncodeToString(hash[:])[:8],
		Topic:        topicSlug,
		Category:     lesson.Category,
		Tier:         tier,
		Summary:      lesson.Summary,
		Mistake:      lesson.Mistake,
		Correction:   lesson.Correction,
		Source:       lesson.Source,
		Confidence:   lesson.Confidence,
		HitCount:     hitCount,
		SupersedesID: supersedesID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if newItem.Confidence <= 0 {
		newItem.Confidence = 5
	}

	if err := store.Save(newItem); err != nil {
		return nil, false, err
	}

	return &newItem, priorMatch != nil, nil
}

func normalizeTopic(topic string) string {
	t := strings.ToLower(strings.TrimSpace(topic))
	t = strings.ReplaceAll(t, " ", "-")
	t = strings.ReplaceAll(t, "_", "-")
	return t
}
