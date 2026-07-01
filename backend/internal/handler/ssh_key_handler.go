package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/ssh"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	"github.com/open-git/backend/internal/repository"
)

type SSHKeyHandler struct {
	keys repository.ISSHKeyStore
}

func NewSSHKeyHandler(keys repository.ISSHKeyStore) *SSHKeyHandler {
	return &SSHKeyHandler{keys: keys}
}

type addSSHKeyRequest struct {
	Title string `json:"title"`
	Key   string `json:"key"`
}

type sshKeyResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   string    `json:"created_at"`
}

func (h *SSHKeyHandler) List(c echo.Context) error {
	userID, err := getUserUUID(c)
	if err != nil {
		return err
	}

	keys, err := h.keys.ListByUserID(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list keys"})
	}

	resp := make([]sshKeyResponse, 0, len(keys))
	for _, key := range keys {
		resp = append(resp, toSSHKeyResponse(key))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *SSHKeyHandler) Add(c echo.Context) error {
	userID, err := getUserUUID(c)
	if err != nil {
		return err
	}

	var req addSSHKeyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}

	if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(req.Key)); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid key format"})
	}

	key := &entity.SSHKey{
		UserID:    userID,
		Title:     req.Title,
		PublicKey: req.Key,
	}
	if err := h.keys.Create(c.Request().Context(), key); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return echo.NewHTTPError(http.StatusConflict, map[string]string{"message": "a key with this fingerprint already exists"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to add key"})
	}

	return c.JSON(http.StatusCreated, toSSHKeyResponse(key))
}

func (h *SSHKeyHandler) Delete(c echo.Context) error {
	userID, err := getUserUUID(c)
	if err != nil {
		return err
	}

	keyID, err := uuid.Parse(c.Param("key_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid key id"})
	}

	if err := h.keys.Delete(c.Request().Context(), keyID, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to delete key"})
	}

	return c.NoContent(http.StatusNoContent)
}

func toSSHKeyResponse(key *entity.SSHKey) sshKeyResponse {
	return sshKeyResponse{
		ID:          key.ID,
		Title:       key.Title,
		Fingerprint: key.Fingerprint,
		CreatedAt:   key.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func getUserUUID(c echo.Context) (uuid.UUID, error) {
	// The auth middleware stores user_id as an int64; convert it the same way
	// the rest of the app does. The previous implementation asserted a
	// uuid.UUID directly, so it always failed (401) and SSH key registration
	// was impossible.
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return uuid.Nil, err
	}
	return middleware.Int64ToUUID(userID), nil
}
