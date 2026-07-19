package handler

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/palemoky/chinese-poetry-api/internal/database"
)

// PoemHandler handles poem-related requests
type PoemHandler struct {
	repo *database.Repository
}

// NewPoemHandler creates a new poem handler
func NewPoemHandler(repo *database.Repository) *PoemHandler {
	return &PoemHandler{
		repo: repo,
	}
}

// ListPoems retrieves a paginated list of poems
// Supports ?lang=zh-Hans (default) or ?lang=zh-Hant
//
// @Summary      诗词列表
// @Description  分页获取诗词列表，支持简繁体切换
// @Tags         诗词
// @Accept       json
// @Produce      json
// @Param        page        query  int     false  "页码"    default(1)
// @Param        page_size   query  int     false  "每页数量"  default(20)
// @Param        lang        query  string  false  "语言 zh-Hans|zh-Hant"  default(zh-Hans)
// @Success      200  {object}  handler.PaginatedResponse
// @Failure      500  {object}  map[string]string
// @Router       /poems [get]
func (h *PoemHandler) ListPoems(c *gin.Context) {
	lang := parseLang(c)
	repo := h.repo.WithLang(lang)
	pagination := ParsePagination(c)

	poems, err := repo.ListPoems(pagination.PageSize, pagination.Offset())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to retrieve poems")
		return
	}

	total, err := repo.CountPoems()
	if err != nil {
		total = 0
	}

	data := make([]map[string]any, len(poems))
	for i, poem := range poems {
		data[i] = formatPoem(&poem)
	}

	c.JSON(http.StatusOK, NewPaginationResponse(data, pagination, int64(total)))
}

// SearchPoems searches for poems by query string
//
// @Summary      搜索诗词
// @Description  根据关键词搜索诗词（支持全文/标题/内容/作者）
// @Tags         诗词
// @Accept       json
// @Produce      json
// @Param        q          query  string  true   "搜索关键词"
// @Param        type       query  string  false  "搜索类型 all|title|content|author"  default(all)
// @Param        page       query  int     false  "页码"      default(1)
// @Param        page_size  query  int     false  "每页数量"   default(20)
// @Param        lang       query  string  false  "语言"      default(zh-Hans)
// @Success      200  {object}  handler.PaginatedResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /poems/search [get]
func (h *PoemHandler) SearchPoems(c *gin.Context) {
	lang := parseLang(c)
	repo := h.repo.WithLang(lang)

	query := c.Query("q")
	if query == "" {
		respondError(c, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	searchType := c.DefaultQuery("type", "all")
	pagination := ParsePagination(c)

	// Use repository's search method instead of search engine
	poems, total, err := repo.SearchPoems(query, searchType, pagination.Page, pagination.PageSize)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "search failed")
		return
	}

	data := make([]map[string]any, len(poems))
	for i, poem := range poems {
		data[i] = formatPoem(&poem)
	}

	c.JSON(http.StatusOK, NewPaginationResponse(data, pagination, total))
}

// filterQueryKeys lists every RandomPoem filter param other than char/lang.
// Used to reject char being combined with them (see RandomPoem doc comment).
var filterQueryKeys = []string{"author_id", "author", "type_id", "type", "dynasty_id", "dynasty"}

// RandomPoem returns a random poem with optional filters
// Supports ?lang=zh-Hans (default) or ?lang=zh-Hant
// Supports filters: ?author=李白&type=五言绝句&type=七言绝句&dynasty=唐
// Or by ID: ?author_id=123&type_id=456&type_id=789&dynasty_id=789
//
// Supports 飞花令-style single-character search: ?char=春
// char is only combinable with lang - not with author/type/dynasty filters,
// since it selects poems via the FTS content index rather than the id-based
// filters used elsewhere in this handler.
//
// @Summary      随机诗词
// @Description  随机返回一首诗词，支持按作者/朝代/类型过滤，以及飞花令单字搜索
// @Tags         诗词
// @Accept       json
// @Produce      json
// @Param        author_id   query  int     false  "作者 ID"
// @Param        author      query  string  false  "作者名"
// @Param        type_id     query  []int   false  "类型 ID（可多个）"
// @Param        type        query  []string false  "类型名（可多个）"
// @Param        dynasty_id  query  int     false  "朝代 ID"
// @Param        dynasty     query  string  false  "朝代名"
// @Param        char        query  string  false  "飞花令：单字搜索（与其它过滤互斥）"
// @Param        lang        query  string  false  "语言"  default(zh-Hans)
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /poems/random [get]
func (h *PoemHandler) RandomPoem(c *gin.Context) {
	lang := parseLang(c)
	repo := h.repo.WithLang(lang)

	if char := c.Query("char"); char != "" {
		for _, key := range filterQueryKeys {
			if c.Query(key) != "" {
				respondError(c, http.StatusBadRequest, "char cannot be combined with author/type/dynasty filters")
				return
			}
		}
		if utf8.RuneCountInString(char) != 1 {
			respondError(c, http.StatusBadRequest, "char must be a single character")
			return
		}

		poem, err := repo.GetRandomPoemByChar(char)
		if err != nil {
			respondError(c, http.StatusNotFound, "no poems found containing the given character")
			return
		}

		c.JSON(http.StatusOK, formatPoem(poem))
		return
	}

	// Parse filter parameters
	var authorID, dynastyID *int64
	var typeIDs []int64

	// Parse author filter (by ID or name)
	if authorIDStr := c.Query("author_id"); authorIDStr != "" {
		if id, err := strconv.ParseInt(authorIDStr, 10, 64); err == nil {
			authorID = &id
		}
	} else if authorName := c.Query("author"); authorName != "" {
		// Look up author by name
		author, err := repo.GetAuthorByName(authorName)
		if err != nil {
			respondError(c, http.StatusNotFound, "author not found")
			return
		}
		authorID = &author.ID
	}

	// Parse type filter (by ID or name) - supports multiple values
	typeIDStrs := c.QueryArray("type_id")
	typeNames := c.QueryArray("type")

	if len(typeIDStrs) > 0 {
		// Parse type IDs
		for _, idStr := range typeIDStrs {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				typeIDs = append(typeIDs, id)
			}
		}
	} else if len(typeNames) > 0 {
		// Batch lookup types by name in a single query
		ids, err := repo.GetPoetryTypeIDs(typeNames)
		if err != nil {
			respondError(c, http.StatusNotFound, "poetry type not found")
			return
		}
		typeIDs = ids
	}

	// Parse dynasty filter (by ID or name)
	if dynastyIDStr := c.Query("dynasty_id"); dynastyIDStr != "" {
		if id, err := strconv.ParseInt(dynastyIDStr, 10, 64); err == nil {
			dynastyID = &id
		}
	} else if dynastyName := c.Query("dynasty"); dynastyName != "" {
		// Look up dynasty by name
		dynasty, err := repo.GetDynastyByName(dynastyName)
		if err != nil {
			respondError(c, http.StatusNotFound, "dynasty not found")
			return
		}
		dynastyID = &dynasty.ID
	}

	// Get a random poem with filters
	poem, err := repo.GetRandomPoem(dynastyID, authorID, typeIDs)
	if err != nil {
		respondError(c, http.StatusNotFound, "no poems found matching the criteria")
		return
	}

	c.JSON(http.StatusOK, formatPoem(poem))
}
