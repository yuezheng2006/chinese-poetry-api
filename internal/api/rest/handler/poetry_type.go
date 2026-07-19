package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/palemoky/chinese-poetry-api/internal/database"
)

// PoetryTypeHandler handles poetry type-related requests
type PoetryTypeHandler struct {
	repo *database.Repository
}

// NewPoetryTypeHandler creates a new poetry type handler
func NewPoetryTypeHandler(repo *database.Repository) *PoetryTypeHandler {
	return &PoetryTypeHandler{repo: repo}
}

// ListPoetryTypes returns a list of poetry types
// Supports ?lang=zh-Hans (default) or ?lang=zh-Hant
//
// @Summary      诗词类型列表
// @Description  获取所有诗词类型（五绝/七绝/五律/七律/宋词/元曲等）及其统计
// @Tags         类型
// @Accept       json
// @Produce      json
// @Param        lang  query  string  false  "语言"  default(zh-Hans)
// @Success      200  {array}  map[string]any
// @Failure      500  {object}  map[string]string
// @Router       /types [get]
func (h *PoetryTypeHandler) ListPoetryTypes(c *gin.Context) {
	lang := parseLang(c)
	repo := h.repo.WithLang(lang)

	types, err := repo.GetPoetryTypesWithStats()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch poetry types")
		return
	}

	data := make([]map[string]any, len(types))
	for i, t := range types {
		data[i] = formatPoetryTypeWithStats(&t)
	}

	respondOK(c, data)
}

// GetPoetryType returns a specific poetry type by ID
// Supports ?lang=zh-Hans (default) or ?lang=zh-Hant
//
// @Summary      诗词类型详情
// @Description  根据 ID 获取诗词类型详细信息
// @Tags         类型
// @Accept       json
// @Produce      json
// @Param        id    path  int     true  "类型 ID"
// @Param        lang  query string  false  "语言"  default(zh-Hans)
// @Success      200  {object}  map[string]any
// @Failure      404  {object}  map[string]string
// @Router       /types/{id} [get]
func (h *PoetryTypeHandler) GetPoetryType(c *gin.Context) {
	lang := parseLang(c)
	repo := h.repo.WithLang(lang)

	id, ok := parseID(c, "id", "poetry type")
	if !ok {
		return
	}

	poetryType, err := repo.GetPoetryTypeByID(id)
	if err != nil {
		respondError(c, http.StatusNotFound, "Poetry type not found")
		return
	}

	respondOK(c, formatPoetryType(poetryType))
}
