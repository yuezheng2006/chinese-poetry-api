package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/palemoky/chinese-poetry-api/internal/database"
)

// AuthorHandler handles author-related requests
type AuthorHandler struct {
	repo *database.Repository
}

// NewAuthorHandler creates a new author handler
func NewAuthorHandler(repo *database.Repository) *AuthorHandler {
	return &AuthorHandler{repo: repo}
}

// ListAuthors returns a list of authors
// Supports ?lang=zh-Hans (default) or ?lang=zh-Hant
//
// @Summary      作者列表
// @Description  分页获取作者列表，按作品数降序排列
// @Tags         作者
// @Accept       json
// @Produce      json
// @Param        page        query  int     false  "页码"    default(1)
// @Param        page_size   query  int     false  "每页数量"  default(20)
// @Param        lang        query  string  false  "语言"    default(zh-Hans)
// @Success      200  {object}  handler.PaginatedResponse
// @Failure      500  {object}  map[string]string
// @Router       /authors [get]
func (h *AuthorHandler) ListAuthors(c *gin.Context) {
	lang := parseLang(c)
	repo := h.repo.WithLang(lang)
	pagination := ParsePagination(c)

	authors, err := repo.GetAuthorsWithStats(pagination.PageSize, pagination.Offset())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch authors")
		return
	}

	total, err := repo.CountAuthors()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to count authors")
		return
	}

	data := make([]map[string]any, len(authors))
	for i, author := range authors {
		data[i] = formatAuthorWithStats(&author)
	}

	c.JSON(http.StatusOK, NewPaginationResponse(data, pagination, int64(total)))
}

// GetAuthor returns a specific author by ID
// Supports ?lang=zh-Hans (default) or ?lang=zh-Hant
//
// @Summary      作者详情
// @Description  根据 ID 获取作者详细信息
// @Tags         作者
// @Accept       json
// @Produce      json
// @Param        id    path  int     true  "作者 ID"
// @Param        lang  query string  false  "语言"  default(zh-Hans)
// @Success      200  {object}  map[string]any
// @Failure      404  {object}  map[string]string
// @Router       /authors/{id} [get]
func (h *AuthorHandler) GetAuthor(c *gin.Context) {
	lang := parseLang(c)
	repo := h.repo.WithLang(lang)

	id, ok := parseID(c, "id", "author")
	if !ok {
		return
	}

	author, err := repo.GetAuthorByID(id)
	if err != nil {
		respondError(c, http.StatusNotFound, "Author not found")
		return
	}

	respondOK(c, formatAuthor(author))
}
