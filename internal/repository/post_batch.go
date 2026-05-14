package repository

import (
	"gin-quickstart/internal/model"

	"gorm.io/gorm"
)

type PostDeleteBatchResult struct {
	Post      *model.Post
	ParentIDs []uint
}

func (r *PostRepository) DeleteTree(id uint64) (*PostDeleteBatchResult, error) {
	var post model.Post
	if err := r.GormDB.Select("id", "thread_id", "author_id", "parent_id").First(&post, id).Error; err != nil {
		return nil, err
	}

	var threadPosts []model.Post
	if err := r.GormDB.Select("id", "parent_id").
		Where("thread_id = ?", post.ThreadID).
		Find(&threadPosts).Error; err != nil {
		return nil, err
	}

	postIDs, parentIDs, ok := collectPostTreeIDs(threadPosts, post.ID)
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}

	tx := r.GormDB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	if err := deletePostsRelatedRows(tx, postIDs); err != nil {
		tx.Rollback()
		return nil, err
	}

	if len(postIDs) > 0 {
		if err := tx.Where("id IN ?", postIDs).Delete(&model.Post{}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &PostDeleteBatchResult{
		Post:      &post,
		ParentIDs: parentIDs,
	}, nil
}
