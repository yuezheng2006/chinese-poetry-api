package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/palemoky/chinese-poetry-api/internal/database"
)

// DynastyHandler handles dynasty-related requests
type DynastyHandler struct {
	repo *database.Repository
}

// NewDynastyHandler creates a new dynasty handler
func NewDynastyHandler(repo *database.Repository) *DynastyHandler {
	return &DynastyHandler{repo: repo}
}

// ListDynasties returns a list of dynasties
// Supports ?lang=zh-Hans (default) or ?lang=zh-Hant
//
// @Summary      朝代列表
// @Description  获取所有朝代及其诗词/作者统计
// @Tags         朝代
// @Accept       json
// @Produce      json
// @Param        lang  query  string  false  "语言"  default(zh-Hans)
// @Success      200  {array}  map[string]any
// @Failure      500  {object}  map[string]string
// @Router       /dynasties [get]
func (h *DynastyHandler) ListDynasties(c *gin.Context) {
	lang := parseLang(c)
	repo := h.repo.WithLang(lang)

	dynasties, err := repo.GetDynastiesWithStats()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch dynasties")
		return
	}

	data := make([]map[string]any, len(dynasties))
	for i, d := range dynasties {
		data[i] = formatDynastyWithStats(&d)
	}

	respondOK(c, data)
}

// GetDynasty returns a specific dynasty by ID
// Supports ?lang=zh-Hans (default) or ?lang=zh-Hant
//
// @Summary      朝代详情
// @Description  根据 ID 获取朝代详细信息
// @Tags         朝代
// @Accept       json
// @Produce      json
// @Param        id    path  int     true  "朝代 ID"
// @Param        lang  query string  false  "语言"  default(zh-Hans)
// @Success      200  {object}  map[string]any
// @Failure      404  {object}  map[string]string
// @Router       /dynasties/{id} [get]
func (h *DynastyHandler) GetDynasty(c *gin.Context) {
	lang := parseLang(c)
	repo := h.repo.WithLang(lang)

	id, ok := parseID(c, "id", "dynasty")
	if !ok {
		return
	}

	dynasty, err := repo.GetDynastyByID(id)
	if err != nil {
		respondError(c, http.StatusNotFound, "Dynasty not found")
		return
	}

	respondOK(c, formatDynasty(dynasty))
}
