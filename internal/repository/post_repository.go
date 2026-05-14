package repository

import (
	"gin-quickstart/internal/model"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type PostRepository struct {
	GormDB      *gorm.DB
	RedisClient *redis.Client
}

func NewPostRepository(db *gorm.DB, redis *redis.Client) *PostRepository {
	return &PostRepository{
		GormDB:      db,
		RedisClient: redis,
	}
}

// GETTER
func (r PostRepository) GetAllPosts() ([]model.Post, error) {
	var posts []model.Post
	err := r.GormDB.
		Preload("Thread").
		Preload("Author").
		Find(&posts).Error

	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r PostRepository) GetPostByID(id uint64) (*model.Post, error) {
	var post model.Post
	err := r.GormDB.
		Preload("Thread").
		Preload("Author").
		Preload("Posts").
		First(&post, id).Error

	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r PostRepository) GetPostsByThreadID(threadID uint64) ([]model.Post, error) {
	var posts []model.Post
	err := r.GormDB.
		Preload("Thread").
		Preload("Author").
		Where("thread_id = ?", threadID).Find(&posts).Error

	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r PostRepository) GetPostsByAuthorID(authorID uint64) ([]model.Post, error) {
	var posts []model.Post
	err := r.GormDB.
		Preload("Thread").
		Preload("Author").
		Where("author_id = ?", authorID).Find(&posts).Error

	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r PostRepository) GetPostsByParentID(parentID uint64) ([]model.Post, error) {
	var posts []model.Post
	err := r.GormDB.
		Preload("Thread").
		Preload("Author").
		Where("parent_id = ?", parentID).Find(&posts).Error

	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r PostRepository) GetPostVotes(postID uint64) ([]model.Vote, error) {
	var votes []model.Vote
	err := r.GormDB.Where("post_id = ?", postID).Find(&votes).Error

	if err != nil {
		return nil, err
	}
	return votes, nil
}

func (r PostRepository) GetVoteByPostAndUserID(postID uint64, userID uint64) (*model.Vote, error) {
	var vote model.Vote
	err := r.GormDB.Where("post_id = ? AND user_id = ?", postID, userID).First(&vote).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &vote, nil
}

func (r PostRepository) GetReactionByPostAndUserID(postID uint64, userID uint64) (*model.Reaction, error) {
	var reaction model.Reaction
	err := r.GormDB.Where("post_id = ? AND user_id = ?", postID, userID).First(&reaction).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &reaction, nil
}

func (r PostRepository) GetThreadByID(id uint64) (*model.Thread, error) {
	var thread model.Thread
	err := r.GormDB.First(&thread, id).Error
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

func (r PostRepository) ThreadHasSolutionExcludingPost(threadID uint, postID uint) (bool, error) {
	var count int64
	err := r.GormDB.
		Model(&model.Post{}).
		Where("thread_id = ? AND id != ? AND is_solution = ?", threadID, postID, true).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SETTER

func (r *PostRepository) Create(post *model.Post) error {
	return r.GormDB.Create(post).Error
}

func (r *PostRepository) Save(post *model.Post) error {
	return r.GormDB.Save(post).Error
}

func (r *PostRepository) Update(post *model.Post) error {
	post.IsEdited = true
	return r.GormDB.Save(post).Error
}

func (r *PostRepository) Delete(post *model.Post) error {
	return r.GormDB.Delete(post).Error
}

func (r *PostRepository) CreateAttachment(postID uint64,
	attachment *model.Attachment) (*model.Attachment, error) {

	r.GormDB.Model(&model.Post{ID: uint(postID)}).Association("Attachments").Append(attachment)

	return attachment, nil
}

func (r *PostRepository) SaveVote(vote *model.Vote) error {
	return r.GormDB.Save(vote).Error
}

func (r *PostRepository) CreateReaction(reaction *model.Reaction) error {
	return r.GormDB.Create(reaction).Error
}

func (r *PostRepository) DeleteReaction(reaction *model.Reaction) error {
	return r.GormDB.Delete(reaction).Error
}

func (r *PostRepository) MarkAsSolution(postID uint64) error {
	return r.GormDB.Model(&model.Post{}).
		Where("id = ?", postID).
		Update("is_solution", true).Error
}
