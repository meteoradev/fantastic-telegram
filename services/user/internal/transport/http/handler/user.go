package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/disdreamq/fantastic-telegram/services/user/internal/domain"
	"github.com/disdreamq/fantastic-telegram/services/user/internal/port"
	"github.com/disdreamq/fantastic-telegram/services/user/internal/service"
	"github.com/go-chi/chi/v5"
)

type UserController struct {
	userService port.UserService
}

type createUserRequest struct {
	Username string `json:"username" example:"johndoe"`
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
}
type updateUserRequest struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" example:"password123"`
}

type userResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// writeError writes a structured JSON error response with the given status code.
func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func NewUserResponse(u *domain.User) *userResponse {
	return &userResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

func NewUserController(userService port.UserService) *UserController {
	return &UserController{userService: userService}

}

// Create handles user registration
// @Summary      Register a new user
// @Description  Creates a new user account
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      createUserRequest  true  "User registration data"
// @Success      201      {object}  userResponse
// @Failure      400      {object}  ErrorResponse  "invalid JSON / failed to create user"
// @Failure      409      {object}  ErrorResponse  "user with this email already exists"
// @Failure      500      {object}  ErrorResponse  "failed to create user"
// @Router       /register [post]
func (c *UserController) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var userReq createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}
	user, err := c.userService.Create(r.Context(), userReq.Username, userReq.Email, userReq.Password)
	if err != nil {
		switch err {
		case service.ErrUnexpected:
			http.Error(w, `{"error": "failed to create user"}`, http.StatusInternalServerError)
			return
		case service.ErrUserAlreadyExists:
			http.Error(w, `{"error": "user with this email already exists"}`, http.StatusConflict)
			return
		default:
			http.Error(w, `{"error": "failed to create user"}`, http.StatusBadRequest)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(NewUserResponse(user))
}

// GetByID retrieves a user by their ID
// @Summary      Get user by ID
// @Description  Returns a single user by their ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        userID  path      int  true  "User ID"
// @Success      200     {object}  userResponse
// @Failure      400     {object}  ErrorResponse  "invalid user ID / failed to get user"
// @Failure      404     {object}  ErrorResponse  "user not found"
// @Router       /users/{userID} [get]
func (c *UserController) GetByID(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid user ID"}`, http.StatusBadRequest)
		return
	}
	user, err := c.userService.GetByID(r.Context(), userID)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
			return
		default:
			http.Error(w, `{"error": "failed to get user"}`, http.StatusBadRequest)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(NewUserResponse(user))
}

// GetByEmail retrieves a user by their email
// @Summary      Get user by email
// @Description  Returns a single user by their email
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        email  path      string  true  "User Email"
// @Success      200    {object}  userResponse
// @Failure      400    {object}  ErrorResponse  "failed to get user"
// @Failure      401    {object}  ErrorResponse  "unauthorized"
// @Failure      404    {object}  ErrorResponse  "user not found"
// @Router       /users/email/{email} [get]
func (c *UserController) GetByEmail(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	email := chi.URLParam(r, "email")
	decodedEmail, err := url.QueryUnescape(email)
	if err != nil {
		http.Error(w, `{"error": "invalid email encoding"}`, http.StatusBadRequest)
		return
	}
	user, err := c.userService.GetByEmail(r.Context(), decodedEmail)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
			return
		default:
			http.Error(w, `{"error": "failed to get user"}`, http.StatusBadRequest)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(NewUserResponse(user))
}

// Update updates an existing user
// @Summary      Update a user
// @Description  Updates an existing user by ID (requires authentication)
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userID    path      int                  true  "User ID"
// @Param        request   body      updateUserRequest    true  "User data to update"
// @Success      200       {string} string               "OK"
// @Failure      400       {object} ErrorResponse        "invalid user ID / invalid JSON"
// @Failure      401       {object} ErrorResponse        "unauthorized"
// @Failure      405     {object} ErrorResponse  "method not allowed"
// @Failure      404       {object} ErrorResponse        "user not found"
// @Failure      500       {object} ErrorResponse        "failed to get user"
// @Router       /users/{userID} [put]
func (c *UserController) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var userReq updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid user ID"}`, http.StatusBadRequest)
		return
	}
	val := r.Context().Value("userID")
	if val == nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var currUserID int64
	switch v := val.(type) {
	case int:
		currUserID = int64(v)
	case int64:
		currUserID = v
	default:
		writeError(w, "invalid user ID type", http.StatusInternalServerError)
		return
	}
	err = c.userService.Update(r.Context(), currUserID, userID, userReq.Username, userReq.Email, userReq.Password)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			writeError(w, "user not found", http.StatusNotFound)
			return
		case service.ErrMethodNotAllowed:
			writeError(w, "failed to update user", http.StatusMethodNotAllowed)
			return
		default:
			writeError(w, "failed to update user", http.StatusBadRequest)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// Delete removes a user by ID
// @Summary      Delete a user
// @Description  Removes a user by ID (requires authentication)
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userID  path      int  true  "User ID"
// @Success      204     {string} string  "No Content"
// @Failure      400     {object} ErrorResponse  "invalid user ID"
// @Failure      401     {object} ErrorResponse  "unauthorized"
// @Failure      404     {object} ErrorResponse  "user not found"
// @Failure      405     {object} ErrorResponse  "method not allowed"
// @Failure      500     {object} ErrorResponse  "failed to get user"
// @Router       /users/{userID} [delete]
func (c *UserController) Delete(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, "invalid user ID", http.StatusBadRequest)
		return
	}
	val := r.Context().Value("userID")
	if val == nil {
		writeError(w, "invalid user ID type", http.StatusInternalServerError)
		return
	}
	var currUserID int64
	switch v := val.(type) {
	case int:
		currUserID = int64(v)
	case int64:
		currUserID = v
	default:
		writeError(w, "invalid user ID type", http.StatusInternalServerError)
		return
	}
	err = c.userService.Delete(r.Context(), currUserID, userID)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			writeError(w, "user not found", http.StatusNotFound)
			return
		case service.ErrMethodNotAllowed:
			writeError(w, "failed to delete user", http.StatusMethodNotAllowed)
			return
		default:
			writeError(w, "failed to delete user", http.StatusBadRequest)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
