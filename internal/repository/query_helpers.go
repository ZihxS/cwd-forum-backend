package repository

import (
	"gin-quickstart/internal/model"

	"gorm.io/gorm"
)

func uniqueUintSlice(values []uint) []uint {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[uint]struct{}, len(values))
	unique := make([]uint, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		unique = append(unique, value)
	}

	return unique
}

func collectPostTreeIDs(posts []model.Post, rootID uint) ([]uint, []uint, bool) {
	if len(posts) == 0 {
		return nil, nil, false
	}

	childrenByParent := make(map[uint][]uint, len(posts))
	parentIDs := make(map[uint]struct{}, len(posts))
	postExists := false

	for _, post := range posts {
		if post.ID == rootID {
			postExists = true
		}

		if post.ParentID != nil {
			childrenByParent[*post.ParentID] = append(childrenByParent[*post.ParentID], post.ID)
			parentIDs[*post.ParentID] = struct{}{}
		}
	}

	if !postExists {
		return nil, nil, false
	}

	queue := []uint{rootID}
	postIDs := make([]uint, 0, len(posts))
	visited := make(map[uint]struct{}, len(posts))

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		if _, exists := visited[currentID]; exists {
			continue
		}

		visited[currentID] = struct{}{}
		postIDs = append(postIDs, currentID)
		queue = append(queue, childrenByParent[currentID]...)
	}

	parentIDList := make([]uint, 0, len(parentIDs))
	for parentID := range parentIDs {
		parentIDList = append(parentIDList, parentID)
	}

	return postIDs, parentIDList, true
}

func deletePostsRelatedRows(tx *gorm.DB, postIDs []uint) error {
	if len(postIDs) == 0 {
		return nil
	}

	if err := tx.Where("post_id IN ?", postIDs).Delete(&model.Attachment{}).Error; err != nil {
		return err
	}

	if err := tx.Where("post_id IN ?", postIDs).Delete(&model.Vote{}).Error; err != nil {
		return err
	}

	if err := tx.Where("post_id IN ?", postIDs).Delete(&model.Reaction{}).Error; err != nil {
		return err
	}

	if err := tx.Where("post_id IN ?", postIDs).Delete(&model.Notification{}).Error; err != nil {
		return err
	}

	return nil
}
