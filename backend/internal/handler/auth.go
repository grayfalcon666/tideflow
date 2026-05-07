package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"tideflow/internal/middleware"
	"tideflow/internal/service"
	"tideflow/pkg/response"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// @Summary 用户注册
// @Description 注册新用户，返回access_token和refresh_token
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "注册信息"
// @Success 201 {object} response.TokenResponse
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=30"`
		Password string `json:"password" binding:"required,min=6,max=128"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, id, err := h.auth.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err == service.ErrUserExists {
			response.Conflict(c, "username already exists")
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Created(c, gin.H{
		"account_id":    id,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}

// @Summary 用户登录
// @Description 使用用户名密码登录，返回access_token和refresh_token
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body LoginRequest true "登录信息"
// @Success 200 {object} response.TokenResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, id, err := h.auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err == service.ErrInvalidPassword {
			response.Unauthorized(c)
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"account_id":    id,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}

// @Summary 刷新令牌
// @Description 使用refresh_token获取新的access_token和refresh_token
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body RefreshRequest true "刷新令牌"
// @Success 200 {object} response.TokenResponse
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, _, err := h.auth.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	response.Success(c, tokens)
}

// @Summary 修改密码
// @Description 用户修改密码，需要提供旧密码验证
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body ChangePasswordRequest true "密码修改信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	err := h.auth.ChangePassword(c.Request.Context(), req.Username, req.OldPassword, req.NewPassword)
	if err != nil {
		if err == service.ErrInvalidPassword {
			response.Unauthorized(c)
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// OAuth2Token handles password grant for Swagger UI login
// @Summary OAuth2 Token (Swagger专用)
// @Description Swagger UI登录专用端点，使用username/password换取access_token
// @Tags 认证
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param username formData string true "用户名"
// @Param password formData string true "密码"
// @Success 200 {object} OAuth2TokenResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/oauth2/token [post]
func (h *AuthHandler) OAuth2Token(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		response.BadRequest(c, "username and password are required")
		return
	}

	tokens, _, err := h.auth.Login(c.Request.Context(), username, password)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokens.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   tokens.ExpiresIn,
	})
}

type OAuth2TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// @Summary 用户登出
// @Description 使当前access_token失效
// @Tags 认证
// @Security OAuth2Password
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if err := h.auth.Logout(c.Request.Context(), userID); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

type UserHandler struct {
	repo *service.UserService
}

func NewUserHandler(repo *service.UserService) *UserHandler {
	return &UserHandler{repo: repo}
}

// @Summary 获取当前用户信息
// @Description 获取已登录用户的信息
// @Tags 用户资料
// @Security OAuth2Password
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.repo.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.Success(c, user)
}

// @Summary 获取指定用户信息
// @Description 通过用户ID获取用户公开信息
// @Tags 用户资料
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	user, err := h.repo.GetUserByID(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.Success(c, user)
}

// @Summary 通过用户名查找用户
// @Description 通过用户名获取用户公开信息
// @Tags 用户资料
// @Produce json
// @Param username path string true "用户名"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/users/username/{username} [get]
func (h *UserHandler) GetUserByUsername(c *gin.Context) {
	username := c.Param("username")
	user, err := h.repo.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.Success(c, user)
}

// @Summary 更新个人资料
// @Description 更新当前用户的头像和简介
// @Tags 用户资料
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param body body UpdateProfileRequest true "更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/users/me [put]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		AvatarURL *string `json:"avatar_url" binding:"omitempty,min=1"`
		Bio       *string `json:"bio" binding:"omitempty,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.AvatarURL != nil {
		updates["avatar_url"] = *req.AvatarURL
	}
	if req.Bio != nil {
		updates["bio"] = *req.Bio
	}

	if err := h.repo.UpdateUser(c.Request.Context(), userID, updates); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// @Summary 修改用户名
// @Description 修改当前用户的用户名
// @Tags 用户资料
// @Security OAuth2Password
// @Accept json
// @Produce json
// @Param body body UpdateUsernameRequest true "新用户名"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/v1/users/me/username [put]
func (h *UserHandler) UpdateUsername(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=30"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.repo.UpdateUsername(c.Request.Context(), userID, req.Username); err != nil {
		if err == service.ErrUserExists {
			response.Conflict(c, "username already exists")
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// @Summary 关注用户
// @Description 当前用户关注指定用户
// @Tags 社交关系
// @Security OAuth2Password
// @Produce json
// @Param user_id path int true "目标用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/users/{user_id}/follow [post]
func (h *UserHandler) Follow(c *gin.Context) {
	userID := middleware.GetUserID(c)
	targetIDStr := c.Param("user_id")
	targetID, err := strconv.ParseUint(targetIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	if err := h.repo.Follow(c.Request.Context(), userID, uint(targetID)); err != nil {
		if errors.Is(err, service.ErrAlreadyFollowing) {
			response.Conflict(c, err.Error())
		} else if strings.Contains(err.Error(), "not found") {
			response.NotFound(c, err.Error())
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}
	response.Success(c, nil)
}

// @Summary 取消关注
// @Description 当前用户取消关注指定用户
// @Tags 社交关系
// @Security OAuth2Password
// @Produce json
// @Param user_id path int true "目标用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/users/{user_id}/follow [delete]
func (h *UserHandler) Unfollow(c *gin.Context) {
	userID := middleware.GetUserID(c)
	targetIDStr := c.Param("user_id")
	targetID, err := strconv.ParseUint(targetIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	if err := h.repo.Unfollow(c.Request.Context(), userID, uint(targetID)); err != nil {
		if errors.Is(err, service.ErrNotFollowing) {
			response.Conflict(c, err.Error())
		} else if strings.Contains(err.Error(), "not found") {
			response.NotFound(c, err.Error())
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}
	response.Success(c, nil)
}

// @Summary 获取关注列表
// @Description 获取指定用户的关注列表
// @Tags 社交关系
// @Security OAuth2Password
// @Produce json
// @Param id path int true "用户ID"
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/users/{id}/following [get]
func (h *UserHandler) GetFollowing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	following, nextCursor, hasMore, err := h.repo.GetFollowing(c.Request.Context(), uint(id), cursor, limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, response.PageResponse{
		Items:      following,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

// @Summary 获取粉丝列表
// @Description 获取指定用户的粉丝列表
// @Tags 社交关系
// @Security OAuth2Password
// @Produce json
// @Param id path int true "用户ID"
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/users/{id}/followers [get]
func (h *UserHandler) GetFollowers(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	followers, nextCursor, hasMore, err := h.repo.GetFollowers(c.Request.Context(), uint(id), cursor, limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, response.PageResponse{
		Items:      followers,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

// @Summary 获取社交数量
// @Description 获取指定用户的关注数和粉丝数
// @Tags 社交关系
// @Security OAuth2Password
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/users/{id}/social-counts [get]
func (h *UserHandler) GetSocialCounts(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	counts, err := h.repo.GetSocialCounts(c.Request.Context(), uint(id))
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, counts)
}

// @Summary 获取用户视频列表
// @Description 获取指定用户发布的视频列表
// @Tags 用户资料
// @Security OAuth2Password
// @Produce json
// @Param id path int true "用户ID"
// @Param cursor query string false "游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} response.PageResponse
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/users/{id}/videos [get]
func (h *UserHandler) GetUserVideos(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	userID := middleware.GetUserID(c)
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	videos, nextCursor, hasMore, err := h.repo.GetUserVideos(c.Request.Context(), uint(id), userID, cursor, limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, response.PageResponse{
		Items:      videos,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

// Request/Response types for swagger
type (
	RegisterRequest struct {
		Username string `json:"username" binding:"required,min=3,max=30"`
		Password string `json:"password" binding:"required,min=6,max=128"`
	}
	LoginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	RefreshRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	ChangePasswordRequest struct {
		Username    string `json:"username" binding:"required"`
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
	}
	UpdateProfileRequest struct {
		AvatarURL *string `json:"avatar_url"`
		Bio       *string `json:"bio"`
	}
	UpdateUsernameRequest struct {
		Username string `json:"username" binding:"required,min=3,max=30"`
	}
)