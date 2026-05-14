package service

import (
	"encoding/json"
	"errors"
	"gin-quickstart/internal/model"
	"gin-quickstart/internal/repository"
	"gin-quickstart/pkg/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type ThreadService struct {
	r *repository.ThreadRepository
}

func NewThreadService(r *repository.ThreadRepository) *ThreadService {
	return &ThreadService{
		r: r,
	}
}

// GETTER
func (s ThreadService) GetAllThreads(ctx *gin.Context) ([]model.Thread, error) {
	getStatus := s.r.RedisClient.Get(ctx, "threads")

	if getStatus.Err() == nil {
		var threads []model.Thread
		err := json.Unmarshal([]byte(getStatus.Val()), &threads)

		if err != nil {
			return nil, err
		}

		return threads, nil
	}

	if getStatus.Err() != nil && getStatus.Err() != redis.Nil {
		return nil, getStatus.Err()
	}

	threads, err := s.r.GetAllThreads()

	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(threads)

	if err != nil {
		return nil, err
	}

	cmdStatus := s.r.RedisClient.Set(ctx, "threads", json, time.Hour)

	if cmdStatus.Err() != nil {
		return nil, cmdStatus.Err()
	}

	return threads, nil
}

func (s ThreadService) GetThreadByID(id uint64, ctx *gin.Context) (*model.Thread, error) {
	getStatus := s.r.RedisClient.Get(ctx, "thread:"+strconv.FormatUint(id, 10))

	if getStatus.Err() == nil {
		var thread model.Thread
		err := json.Unmarshal([]byte(getStatus.Val()), &thread)

		if err != nil {
			return nil, err
		}

		return &thread, nil
	}

	if getStatus.Err() != nil && getStatus.Err() != redis.Nil {
		return nil, getStatus.Err()
	}

	thread, err := s.r.GetThreadByID(id)

	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(thread)

	if err != nil {
		return nil, err
	}

	cmdStatus := s.r.RedisClient.Set(ctx, "thread:"+strconv.FormatUint(id, 10), json, time.Hour)

	if cmdStatus.Err() != nil {
		return nil, cmdStatus.Err()
	}

	return thread, nil
}

func (s ThreadService) GetThreadBySlug(slug string, ctx *gin.Context) (*model.Thread, error) {
	getStatus := s.r.RedisClient.Get(ctx, "thread:slug:"+slug)

	if getStatus.Err() == nil {
		var thread model.Thread
		err := json.Unmarshal([]byte(getStatus.Val()), &thread)

		if err != nil {
			return nil, err
		}

		return &thread, nil
	}

	if getStatus.Err() != nil && getStatus.Err() != redis.Nil {
		return nil, getStatus.Err()
	}

	thread, err := s.r.GetThreadBySlug(slug)

	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(thread)

	if err != nil {
		return nil, err
	}

	cmdStatus := s.r.RedisClient.Set(ctx, "thread:slug:"+slug, json, time.Hour)

	if cmdStatus.Err() != nil {
		return nil, cmdStatus.Err()
	}

	return thread, nil
}

func (s ThreadService) GetThreadsByCategoryID(categoryID uint, ctx *gin.Context) ([]model.Thread, error) {
	getStatus := s.r.RedisClient.Get(ctx, "threads:category:"+strconv.FormatUint(uint64(categoryID), 10))

	if getStatus.Err() == nil {
		var threads []model.Thread
		err := json.Unmarshal([]byte(getStatus.Val()), &threads)

		if err != nil {
			return nil, err
		}

		return threads, nil
	}

	if getStatus.Err() != nil && getStatus.Err() != redis.Nil {
		return nil, getStatus.Err()
	}

	threads, err := s.r.GetThreadsByCategoryID(categoryID)

	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(threads)

	if err != nil {
		return nil, err
	}

	cmdStatus := s.r.RedisClient.Set(ctx, "threads:category:"+strconv.FormatUint(uint64(categoryID), 10), json, time.Hour)

	if cmdStatus.Err() != nil {
		return nil, cmdStatus.Err()
	}

	return threads, nil
}

func (s ThreadService) GetThreadsByAuthorID(authorID uint, ctx *gin.Context) ([]model.Thread, error) {
	getStatus := s.r.RedisClient.Get(ctx, "threads:author:"+strconv.FormatUint(uint64(authorID), 10))

	if getStatus.Err() == nil {
		var threads []model.Thread
		err := json.Unmarshal([]byte(getStatus.Val()), &threads)

		if err != nil {
			return nil, err
		}

		return threads, nil
	}

	if getStatus.Err() != nil && getStatus.Err() != redis.Nil {
		return nil, getStatus.Err()
	}

	threads, err := s.r.GetThreadsByAuthorID(authorID)

	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(threads)

	if err != nil {
		return nil, err
	}

	cmdStatus := s.r.RedisClient.Set(ctx, "threads:author:"+strconv.FormatUint(uint64(authorID), 10), json, time.Hour)

	if cmdStatus.Err() != nil {
		return nil, cmdStatus.Err()
	}

	return threads, nil
}

func (s ThreadService) GetThreadsByTagID(tagID uint, ctx *gin.Context) ([]model.Thread, error) {
	getStatus := s.r.RedisClient.Get(ctx, "threads:tag:"+strconv.FormatUint(uint64(tagID), 10))

	if getStatus.Err() == nil {
		var threads []model.Thread
		err := json.Unmarshal([]byte(getStatus.Val()), &threads)

		if err != nil {
			return nil, err
		}

		return threads, nil
	}

	if getStatus.Err() != nil && getStatus.Err() != redis.Nil {
		return nil, getStatus.Err()
	}

	threads, err := s.r.GetThreadsByTagID(tagID)

	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(threads)

	if err != nil {
		return nil, err
	}

	cmdStatus := s.r.RedisClient.Set(ctx, "threads:tag:"+strconv.FormatUint(uint64(tagID), 10), json, time.Hour)

	if cmdStatus.Err() != nil {
		return nil, cmdStatus.Err()
	}

	return threads, nil
}

// SETTER
func (s *ThreadService) Create(
	CategoryID uint,
	Title string,
	Slug string,
	Content string,
	AuthorID uint,
	TagIDs []uint,
	ctx *gin.Context,
) (*model.Thread, *model.Post, error) {
	thread := &model.Thread{
		CategoryID: CategoryID,
		Title:      Title,
		Slug:       Slug,
		AuthorID:   AuthorID,
	}

	userExists, err := s.r.AuthorExists(AuthorID)
	if err != nil {
		return nil, nil, err
	}

	if !userExists {
		return nil, nil, errors.New("Author is not found!")
	}

	slugExists, _ := s.r.GetThreadBySlug(Slug)

	if slugExists != nil {
		thread.Slug = Slug + "-" + utils.String(5)
	}

	var post *model.Post

	if Content != "" {
		post = &model.Post{
			Content:  Content,
			AuthorID: AuthorID,
		}
	}

	result, err := s.r.CreateWithPostAndTags(thread, post, TagIDs)
	if err != nil {
		return nil, nil, err
	}

	thread = result.Thread
	post = result.Post

	cacheKeys := []string{
		"threads",
		"thread:" + strconv.FormatUint(uint64(thread.ID), 10),
		"thread:slug:" + thread.Slug,
	}

	if post != nil {
		cacheKeys = append(cacheKeys, "posts", "post:id:"+strconv.FormatUint(uint64(post.ID), 10))
	}

	for _, tag := range result.Tags {
		cacheKeys = append(cacheKeys, "tag:id:"+strconv.FormatUint(uint64(tag.ID), 10))
		cacheKeys = append(cacheKeys, "tag:slug:"+tag.Slug)
	}

	delStatus := s.r.RedisClient.Del(ctx, cacheKeys...)

	if delStatus.Err() != nil {
		return thread, post, delStatus.Err()
	}

	return thread, post, nil
}

func (s *ThreadService) Update(
	ID uint64,
	CategoryID *uint,
	Title *string,
	Slug *string,
	IsPinned *bool,
	IsLocked *bool,
	IsSolved *bool,
	ctx *gin.Context,
) (*model.Thread, error) {
	thread, err := s.r.GetThreadByID(ID)

	if err != nil {
		return nil, err
	}

	if thread == nil {
		return nil, errors.New("Thread not found")
	}

	if CategoryID != nil {
		thread.CategoryID = *CategoryID
	}

	if Title != nil {
		thread.Title = *Title
	}

	if Slug != nil {
		slugExists, _ := s.r.GetThreadBySlug(*Slug)

		if slugExists != nil && uint64(slugExists.ID) != ID {
			var newSlug string

			newSlug = *Slug + "-" + utils.String(5)

			Slug = &newSlug
		}

		thread.Slug = *Slug
	}

	if IsPinned != nil {
		thread.IsPinned = *IsPinned
	}

	if IsLocked != nil {
		thread.IsLocked = *IsLocked
	}

	if IsSolved != nil {
		thread.IsSolved = *IsSolved
	}

	err = s.r.Update(thread)

	if err != nil {
		return nil, err
	}

	delStatus := s.r.RedisClient.Del(ctx, "threads", "thread:"+strconv.FormatUint(uint64(thread.ID), 10), "thread:slug:"+thread.Slug)

	if delStatus.Err() != nil {
		return nil, delStatus.Err()
	}

	return thread, nil
}

func (s *ThreadService) Delete(ID uint64, ctx *gin.Context) error {
	thread, err := s.r.DeleteCascade(ID)
	if err != nil {
		return err
	}

	delThreadCacheStatus := s.r.RedisClient.Del(ctx, "threads", "thread:"+strconv.FormatUint(uint64(thread.ID), 10), "thread:slug:"+thread.Slug)

	if delThreadCacheStatus.Err() != nil {
		return delThreadCacheStatus.Err()
	}

	return nil
}

func (s *ThreadService) CreatePostAttachment(post *model.Post, attachment *model.Attachment, ctx *gin.Context) error {
	s.r.RedisClient.Del(ctx, "attachments")
	return s.r.CreatePostAttachment(post, attachment)
}

func (s *ThreadService) CanMarkAsSolution(threadID uint64, userID uint64, ctx *gin.Context) (bool, error) {
	thread, err := s.r.GetThreadByID(threadID)

	if err != nil {
		return false, err
	}

	if thread == nil {
		return false, errors.New("Thread not found")
	}

	if thread.AuthorID != uint(userID) {
		return false, errors.New("Unauthorized")
	}

	return true, nil
}
