// Package handler Swagger documentation for the post service
// @title          Post Service API
// @version        1.0
// @description    REST API для управления постами блога
// @host           localhost:8081
// @BasePath       /
// @schemes        http
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/disdreamq/fantastic-telegram/services/post/internal/domain"
	"github.com/disdreamq/fantastic-telegram/services/post/internal/port"
	"github.com/disdreamq/fantastic-telegram/services/post/internal/service"
	"github.com/go-chi/chi/v5"
)

type PostController struct {
	postService port.PostService
}

type createPostRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type updatePostRequest struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type postResponse struct {
	ID        int64       `json:"id"`
	UserID    int64       `json:"user_id"`
	Title     string      `json:"title"`
	Content   string      `json:"content"`
	CreatedAt interface{} `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func newPostResponse(p *domain.Post) *postResponse {
	return &postResponse{ID: p.ID, UserID: p.UserID, Title: p.Title, Content: p.Content, CreatedAt: p.CreatedAt}
}

// writeError writes a structured JSON error response with the given status code.
func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func NewPostController(postService port.PostService) *PostController {
	return &PostController{postService: postService}

}

// Create handles creating a new post
// @Summary      Create a new post
// @Description  Creates a new post for the authenticated user (requires authentication)
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      createPostRequest  true  "Post data"
// @Success      201      {object}  postResponse
// @Failure      400      {object} ErrorResponse  "invalid JSON / invalid title or content"
// @Failure      401      {object} ErrorResponse  "unauthorized"
// @Failure      409      {object} ErrorResponse  "linked user with this id doesnt exists"
// @Failure      500      {object} ErrorResponse  "failed to create post"
// @Router       /posts/ [post]
func (c *PostController) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var postReq createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&postReq); err != nil {
		writeError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if postReq.Title == "" {
		writeError(w, "title must contain at least 1 character", http.StatusBadRequest)
		return
	}
	if postReq.Content == "" {
		writeError(w, "content must contain at least 1 character", http.StatusBadRequest)
		return
	}
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	post, err := c.postService.Create(r.Context(), userID, postReq.Title, postReq.Content)
	if err != nil {
		switch err {
		case service.ErrUnexpected:
			writeError(w, "failed to create post", http.StatusInternalServerError)
			return
		case service.ErrLinkedUserNotFound:
			writeError(w, "linked user with this id doesnt exists.", http.StatusConflict)
			return
		case domain.ErrInvalidTitle:
			writeError(w, "title must contain at least 1 character", http.StatusBadRequest)
			return
		case domain.ErrInvalidContent:
			writeError(w, "content must contain at least 1 character", http.StatusBadRequest)
			return
		default:
			writeError(w, "failed to create post", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newPostResponse(post))
}

// GetByID retrieves a post by its ID
// @Summary      Get post by ID
// @Description  Returns a single post by its ID (public endpoint, no authentication required)
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        postID  path      int  true  "Post ID"
// @Success      200     {object}  postResponse
// @Failure      400     {object} ErrorResponse  "invalid post ID"
// @Failure      404     {object} ErrorResponse  "post not found"
// @Failure      500     {object} ErrorResponse  "failed to get post"
// @Router       /posts/id/{postID} [get]
func (c *PostController) GetByID(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64)
	if err != nil {
		writeError(w, "invalid post ID", http.StatusBadRequest)
		return
	}
	post, err := c.postService.GetByID(r.Context(), postID)
	if err != nil {
		switch err {
		case service.ErrPostNotFound:
			writeError(w, "post not found", http.StatusNotFound)
			return
		default:
			writeError(w, "failed to get post", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newPostResponse(post))
}

// GetByTitle retrieves a post by its title
// @Summary      Get post by title
// @Description  Returns a single post by its title (public endpoint, no authentication required)
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        title   path      string  true  "Post Title"
// @Success      200     {object}  postResponse
// @Failure      400     {object} ErrorResponse  "invalid post title / failed to get post"
// @Failure      404     {object} ErrorResponse  "post not found"
// @Router       /posts/title/{title} [get]
func (c *PostController) GetByTitle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	title := chi.URLParam(r, "title")
	decodedTitle, err := url.QueryUnescape(title)
	if err != nil {
		writeError(w, "invalid title encoding", http.StatusBadRequest)
		return
	}
	if decodedTitle == "" {
		writeError(w, "invalid post title", http.StatusBadRequest)
		return
	}
	post, err := c.postService.GetByTitle(r.Context(), decodedTitle)
	if err != nil {
		switch err {
		case service.ErrPostNotFound:
			writeError(w, "post not found", http.StatusNotFound)
			return
		default:
			writeError(w, "failed to get post", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newPostResponse(post))
}

// Update updates an existing post
// @Summary      Update a post
// @Description  Updates an existing post by ID (requires authentication)
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        postID    path      int                true  "Post ID"
// @Param        request   body      updatePostRequest  true  "Post data to update"
// @Success      200       {string} string             "OK"
// @Failure      400       {object} ErrorResponse      "invalid post ID / invalid JSON / invalid user ID"
// @Failure      401       {object} ErrorResponse      "unauthorized"
// @Failure      404       {object} ErrorResponse      "user not found"
// @Failure      500       {object} ErrorResponse      "failed to get post"
// @Router       /posts/{postID} [put]
func (c *PostController) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var postReq updatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&postReq); err != nil {
		writeError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	currUserID, ok := r.Context().Value("userID").(int64)
	if !ok {
		writeError(w, "invalid user ID", http.StatusBadRequest)
		return
	}
	err := c.postService.UpdateWithValidate(r.Context(), currUserID, postReq.ID, postReq.Title, postReq.Content)
	if err != nil {
		switch err {
		case service.ErrPostNotFound:
			writeError(w, "post not found", http.StatusNotFound)
			return
		default:
			writeError(w, "failed to update post", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// Delete removes a post by ID
// @Summary      Delete a post
// @Description  Removes a post by ID (requires authentication)
// @Tags         posts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        postID  path      int  true  "Post ID"
// @Success      204     {string} string  "No Content"
// @Failure      400     {object} ErrorResponse  "invalid post ID / invalid user ID"
// @Failure      401     {object} ErrorResponse  "unauthorized"
// @Failure      404     {object} ErrorResponse  "post not found"
// @Failure      500     {object} ErrorResponse  "failed to get post"
// @Router       /posts/{postID} [delete]
func (c *PostController) Delete(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64)
	if err != nil {
		writeError(w, "invalid post ID", http.StatusBadRequest)
		return
	}

	currUserID, ok := r.Context().Value("userID").(int64)
	if !ok {
		writeError(w, "invalid user ID", http.StatusBadRequest)
		return
	}

	err = c.postService.DeleteWithValidate(r.Context(), currUserID, postID)
	if err != nil {
		switch err {
		case service.ErrPostNotFound:
			writeError(w, "post not found", http.StatusNotFound)
			return
		default:
			writeError(w, "failed to delete post", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
