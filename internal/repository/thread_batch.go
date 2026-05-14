package repository

import (
	"errors"
	"gin-quickstart/internal/model"
)

type ThreadCreateBatchResult struct {
	Thread *model.Thread
	Post   *model.Post
	Tags   []model.Tag
}

func (r *ThreadRepository) AuthorExists(authorID uint) (bool, error) {
	var userExists bool

	err := r.GormDB.
		Model(&model.User{}).
		Where("id = ?", authorID).
		Select("count(*) > 0").
		Row().
		Scan(&userExists)

	if err != nil {
		return false, err
	}

	return userExists, nil
}

func (r *ThreadRepository) CreateWithPostAndTags(
	thread *model.Thread,
	post *model.Post,
	tagIDs []uint,
) (*ThreadCreateBatchResult, error) {
	tx := r.GormDB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	if err := tx.Create(thread).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if post != nil {
		post.ThreadID = thread.ID

		if err := tx.Create(post).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	result := &ThreadCreateBatchResult{
		Thread: thread,
		Post:   post,
	}

	if len(tagIDs) > 0 {
		uniqueTagIDs := uniqueUintSlice(tagIDs)
		var tags []model.Tag

		if err := tx.Where("id IN ?", uniqueTagIDs).Find(&tags).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if len(tags) != len(uniqueTagIDs) {
			tx.Rollback()
			return nil, errors.New("one or more tags not found")
		}

		if err := tx.Model(thread).Association("Tags").Append(&tags); err != nil {
			tx.Rollback()
			return nil, err
		}

		result.Tags = tags
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *ThreadRepository) DeleteCascade(id uint64) (*model.Thread, error) {
	var thread model.Thread
	if err := r.GormDB.Select("id", "slug").First(&thread, id).Error; err != nil {
		return nil, err
	}

	tx := r.GormDB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	var postIDs []uint
	if err := tx.Model(&model.Post{}).Where("thread_id = ?", thread.ID).Pluck("id", &postIDs).Error; err != nil {
		tx.Rollback()
		return nil, err
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

	if err := tx.Model(&thread).Association("Tags").Clear(); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Delete(&thread).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &thread, nil
}
