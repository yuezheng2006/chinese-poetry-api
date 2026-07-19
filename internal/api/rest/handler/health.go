package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/palemoky/chinese-poetry-api/internal/database"
)

// HealthHandler handles health check requests
//
// @Summary      健康检查
// @Description  检查 API 服务及数据库连接状态
// @Tags         基础
// @Produce      json
// @Success      200  {object}  map[string]string  "healthy"
// @Success      503  {object}  map[string]string  "unhealthy"
// @Router       /health [get]
func HealthHandler(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check database connection
		sqlDB, err := db.DB.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "failed to get database connection",
			})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	}
}

// StatsHandler returns overall statistics
//
// @Summary      统计信息
// @Description  获取诗词数据库的统计信息（总数、按朝代/类型分布）
// @Tags         基础
// @Produce      json
// @Success      200  {object}  database.Statistics
// @Failure      500  {object}  map[string]string
// @Router       /stats [get]
func StatsHandler(repo *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := repo.GetStatistics()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to get statistics",
			})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}
