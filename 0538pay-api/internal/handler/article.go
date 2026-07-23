package handler

import (
	"strconv"

	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// ArticleHandler 文章 + 分类管理（对齐 epay admin/article.php pre_article CRUD）+ 官网公开读取。
type ArticleHandler struct {
	svc *service.ArticleService
}

func NewArticleHandler(svc *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{svc: svc}
}

// ===== 官网公开读取 =====

// Public GET /api/site/articles 官网首页/文章页：分类 + 显示中的文章（前端按分类分列组装）。
func (h *ArticleHandler) Public(c *gin.Context) {
	cats, err := h.svc.ListCategories()
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	arts, err := h.svc.PublicList()
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"categories": cats, "articles": arts})
}

// PublicDetail GET /api/site/articles/:id 文章详情（浏览量 +1，对齐 epay ?mod=article count+1）。
func (h *ArticleHandler) PublicDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	v, err := h.svc.Detail(uint(id), true)
	if err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, v)
}

// ===== 后台管理 =====

// List GET /api/admin/articles 后台文章列表（DB 分页 + 标题搜索）。
func (h *ArticleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	kw := c.Query("kw")
	list, total, err := h.svc.List(kw, page, pageSize)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list, "total": total})
}

// Create POST /api/admin/articles 新增文章
func (h *ArticleHandler) Create(c *gin.Context) {
	var req service.ArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	id, err := h.svc.Create(req)
	if err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// Update PUT /api/admin/articles/:id 编辑文章
func (h *ArticleHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req service.ArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(uint(id), req); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// SetActive PUT /api/admin/articles/:id/status 显隐切换（对齐 epay ajax setActive）。
func (h *ArticleHandler) SetActive(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetActive(uint(id), req.Status); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// Delete DELETE /api/admin/articles/:id 删除文章
func (h *ArticleHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svc.Delete(uint(id)); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// ===== 分类管理 =====

// ListCategories GET /api/admin/article-categories 分类列表
func (h *ArticleHandler) ListCategories(c *gin.Context) {
	list, err := h.svc.ListCategories()
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// CreateCategory POST /api/admin/article-categories 新增分类
func (h *ArticleHandler) CreateCategory(c *gin.Context) {
	var req service.CategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	id, err := h.svc.CreateCategory(req)
	if err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// UpdateCategory PUT /api/admin/article-categories/:id 编辑分类
func (h *ArticleHandler) UpdateCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req service.CategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.UpdateCategory(uint(id), req); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// DeleteCategory DELETE /api/admin/article-categories/:id 删除分类（级联删该分类下文章）
func (h *ArticleHandler) DeleteCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svc.DeleteCategory(uint(id)); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}
