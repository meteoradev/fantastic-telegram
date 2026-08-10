package post

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/disdreamq/fantastic-telegram/services/post/internal/domain"
	"github.com/disdreamq/fantastic-telegram/services/post/internal/port"
	"github.com/disdreamq/fantastic-telegram/services/post/internal/service"
)

type PostService struct {
	postRepo port.PostRepository
	cache    port.Cache
}

func NewPostService(postRepo port.PostRepository, cache port.Cache) *PostService {
	return &PostService{postRepo: postRepo, cache: cache}
}

func (p *PostService) Create(ctx context.Context, userID int64, title, content string) (*domain.Post, error) {
	domainPost, err := domain.NewPost(userID, title, content)
	if err != nil {
		service.LogError(ctx, "Create post error in domain validation", "user_id", userID, "title", title, "error", err)
		return nil, err
	}
	post, err := p.postRepo.Create(ctx, domainPost)
	if err != nil {
		service.LogError(ctx, "Create post error in repository", "user_id", userID, "title", title, "error", err)
		return nil, service.ErrLinkedUserNotFound
	}
	data, err := json.Marshal(post)
	if err != nil {
		service.LogError(ctx, "JSON marshal error in Create", "user_id", userID, "error", err)
		return nil, err
	}
	p.cache.Set(ctx, "post_"+strconv.FormatInt(post.ID, 10), data, 10*time.Minute)

	service.LogInfo(ctx, "Created post", "user_id", userID, "title", title)
	return post, nil
}

func (p *PostService) GetByID(ctx context.Context, postID int64) (*domain.Post, error) {
	cachedPost, ok := p.cache.Get(ctx, "post_"+strconv.FormatInt(postID, 10))
	if !ok {
		post, err := p.postRepo.GetByID(ctx, postID)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				service.LogDebug(ctx, "Post not found in DB", "post_id", postID, "error", err)
				return nil, service.ErrPostNotFound
			default:
				service.LogError(ctx, "Read post by ID error", "post_id", postID, "error", err)
				return nil, service.ErrUnexpected
			}
		}
		data, err := json.Marshal(post)
		if err != nil {
			service.LogError(ctx, "JSON marshal error in GetByID", "post_id", postID, "error", err)
			return nil, err
		}
		p.cache.Set(ctx, "post_"+strconv.FormatInt(postID, 10), data, 10*time.Minute)
		return post, nil
	}

	var post domain.Post
	err := json.Unmarshal([]byte(cachedPost), &post)
	if err != nil {
		service.LogError(ctx, "Cache unmarshal error in GetByID", "post_id", postID, "error", err)
		return nil, service.ErrCacheUnmarshal
	}

	service.LogDebug(ctx, "Read post from cache", "post_id", postID)
	return &post, nil
}

func (p *PostService) GetByTitle(ctx context.Context, title string) (*domain.Post, error) {
	cachedPost, ok := p.cache.Get(ctx, "post_"+title)
	if !ok {
		post, err := p.postRepo.GetByTitle(ctx, title)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				service.LogDebug(ctx, "Post not found by title", "title", title, "error", err)
				return nil, service.ErrPostNotFound
			default:
				service.LogError(ctx, "Read post by title error", "title", title, "error", err)
				return nil, service.ErrUnexpected
			}
		}
		data, err := json.Marshal(post)
		if err != nil {
			service.LogError(ctx, "JSON marshal error in GetByTitle", "title", title, "error", err)
			return nil, err
		}
		p.cache.Set(ctx, "post_"+title, data, 10*time.Minute)
		return post, nil
	}

	var post domain.Post
	err := json.Unmarshal([]byte(cachedPost), &post)
	if err != nil {
		service.LogError(ctx, "Cache unmarshal error in GetByTitle", "title", title, "error", err)
		return nil, service.ErrCacheUnmarshal
	}

	service.LogDebug(ctx, "Read post by title from cache", "title", title)
	return &post, nil
}

func (p *PostService) UpdateWithValidate(ctx context.Context, currUserID, postID int64, title, content string) error {
	post, err := domain.NewPost(currUserID, title, content)
	if err != nil {
		service.LogError(ctx, "Update post with validate domain error", "post_id", postID, "error", err)
		return err
	}
	post.ID = postID
	err = p.postRepo.UpdateWithValidate(ctx, currUserID, post)
	if err != nil {
		service.LogError(ctx, "Update post with validate repository error", "post_id", postID, "error", err)
		switch err {
		case sql.ErrNoRows:
			return service.ErrUpdatePostFailed
		default:
			return service.ErrUnexpected
		}
	}

	p.cache.Del(ctx, "post_"+strconv.FormatInt(postID, 10))
	p.cache.Del(ctx, "post_"+title)
	service.LogDebug(ctx, "Updated post", "post_id", postID)
	return nil
}

func (p *PostService) Update(ctx context.Context, postID int64, title, content string) error {
	post, err := domain.NewPost(0, title, content)
	if err != nil {
		service.LogError(ctx, "Update post domain error", "post_id", postID, "error", err)
		return err
	}
	post.ID = postID
	err = p.postRepo.Update(ctx, post)
	if err != nil {
		service.LogError(ctx, "Update post repository error", "post_id", postID, "error", err)
		switch err {
		case sql.ErrNoRows:
			return service.ErrPostNotFound
		default:
			return service.ErrUnexpected
		}
	}

	p.cache.Del(ctx, "post_"+strconv.FormatInt(postID, 10))
	p.cache.Del(ctx, "post_"+title)
	service.LogDebug(ctx, "Updated post", "post_id", postID)
	return nil
}

func (p *PostService) Delete(ctx context.Context, postID int64) error {
	title, err := p.postRepo.Delete(ctx, postID)
	if err != nil {
		service.LogError(ctx, "Delete post repository error", "post_id", postID, "error", err)
		switch err {
		case sql.ErrNoRows:
			return service.ErrPostNotFound
		default:
			return service.ErrUnexpected
		}
	}

	p.cache.Del(ctx, "post_"+strconv.FormatInt(postID, 10))
	p.cache.Del(ctx, "post_"+title)
	service.LogDebug(ctx, "Deleted post", "post_id", postID)
	return nil
}

func (p *PostService) DeleteWithValidate(ctx context.Context, currUserID, postID int64) error {
	title, err := p.postRepo.DeleteWithValidate(ctx, currUserID, postID)
	if err != nil {
		service.LogError(ctx, "Delete post with validate repository error", "post_id", postID, "error", err)
		switch err {
		case sql.ErrNoRows:
			return service.ErrDeletePostFailed
		default:
			return service.ErrUnexpected
		}
	}

	p.cache.Del(ctx, "post_"+strconv.FormatInt(postID, 10))
	p.cache.Del(ctx, "post_"+title)
	service.LogDebug(ctx, "Deleted post with validate", "post_id", postID)
	return nil
}
