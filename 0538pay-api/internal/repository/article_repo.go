package repository

import (
	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// ArticleRepo 文章 + 分类数据访问（对齐 epay article.php pre_article 行表 CRUD）。
type ArticleRepo struct{ db *gorm.DB }

func NewArticleRepo(db *gorm.DB) *ArticleRepo { return &ArticleRepo{db: db} }

// ===== 分类（我方官网扩展）=====

// ListCategories 全部分类，按 sort 升序。
func (r *ArticleRepo) ListCategories() ([]model.ArticleCategory, error) {
	var list []model.ArticleCategory
	err := r.db.Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

// FindCategory 按 id 查分类。未找到返回 (nil,nil)。
func (r *ArticleRepo) FindCategory(id uint) (*model.ArticleCategory, error) {
	var c model.ArticleCategory
	err := r.db.Where("id = ?", id).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ArticleRepo) CreateCategory(c *model.ArticleCategory) error { return r.db.Create(c).Error }

func (r *ArticleRepo) UpdateCategory(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.ArticleCategory{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteCategory 删除分类，并级联删除该分类下的文章（对齐前端 removeCategory 语义）。
func (r *ArticleRepo) DeleteCategory(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("category_id = ?", id).Delete(&model.Article{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.ArticleCategory{}, id).Error
	})
}

// ===== 文章（对齐 epay pre_article CRUD）=====

// List 分页查询文章（DB 侧分页 + title LIKE 搜索，对齐 epay article.php）。
// kw 非空按标题模糊；onlyActive=true 仅取显示中的（官网公开读用）。
func (r *ArticleRepo) List(kw string, onlyActive bool, offset, limit int) ([]model.Article, int64, error) {
	tx := r.db.Model(&model.Article{})
	if kw != "" {
		tx = tx.Where("title LIKE ?", "%"+kw+"%")
	}
	if onlyActive {
		tx = tx.Where("active = 1")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Article
	// 置顶优先 → sort 升序 → id 倒序（对齐 epay 列表 order by id desc + 置顶标）
	err := tx.Order("top DESC, sort ASC, id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

// ListAllActive 全部显示中的文章（官网首页按分类分列用，量小不分页）。
func (r *ArticleRepo) ListAllActive() ([]model.Article, error) {
	var list []model.Article
	err := r.db.Where("active = 1").Order("top DESC, sort ASC, id DESC").Find(&list).Error
	return list, err
}

// FindArticle 按 id 查文章。未找到返回 (nil,nil)。
func (r *ArticleRepo) FindArticle(id uint) (*model.Article, error) {
	var a model.Article
	err := r.db.Where("id = ?", id).First(&a).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ArticleRepo) CreateArticle(a *model.Article) error { return r.db.Create(a).Error }

func (r *ArticleRepo) UpdateArticle(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).Updates(fields).Error
}

func (r *ArticleRepo) DeleteArticle(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.Article{}).Error
}

// SetActive 显示/隐藏切换（对齐 epay ajax.php setActive）。
func (r *ArticleRepo) SetActive(id uint, active int8) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).Update("active", active).Error
}

// IncrCount 浏览量原子 +1（对齐 epay 详情页 ?mod=article 查看时 count+1）。
func (r *ArticleRepo) IncrCount(id uint) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("count", gorm.Expr("count + 1")).Error
}

// CountByTitle 标题唯一性校验（对齐 epay 添加时标题已存在拒）。excludeID>0 排除自身。
func (r *ArticleRepo) CountByTitle(title string, excludeID uint) (int64, error) {
	tx := r.db.Model(&model.Article{}).Where("title = ?", title)
	if excludeID > 0 {
		tx = tx.Where("id != ?", excludeID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// CountArticles 文章总数（seed 判空用）。
func (r *ArticleRepo) CountArticles() (int64, error) {
	var n int64
	err := r.db.Model(&model.Article{}).Count(&n).Error
	return n, err
}
