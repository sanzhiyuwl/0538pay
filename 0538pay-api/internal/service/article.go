package service

import (
	"strings"
	"time"

	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
)

// ArticleService 文章 + 分类业务（对齐 epay article.php pre_article 行表 CRUD）。
// 后台运营编辑 → 官网首页「最新动态」板块 + 文章详情页读取。
type ArticleService struct {
	repo *repository.ArticleRepo
}

func NewArticleService(repo *repository.ArticleRepo) *ArticleService {
	return &ArticleService{repo: repo}
}

// ArticleError 业务错误。
type ArticleError struct{ Msg string }

func (e *ArticleError) Error() string { return e.Msg }

func artErr(msg string) *ArticleError { return &ArticleError{Msg: msg} }

const articleDateLayout = "2006-01-02" // 对齐前端 mock addtime 格式

// ===== 对外视图（字段/json 对齐前端 mock/articles.ts）=====

// CategoryView 分类对外响应。
type CategoryView struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	EnName string `json:"enName"`
	Cover  string `json:"cover"`
	Sort   int    `json:"sort"`
}

// ArticleView 文章对外响应（对齐前端 Article）。
type ArticleView struct {
	ID         uint   `json:"id"`
	CategoryID uint   `json:"categoryId"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Content    string `json:"content"`
	Cover      string   `json:"cover"`
	Tags       []string `json:"tags"` // 文章标签数组（存储为逗号分隔串，对外拆成数组）
	IsNew      bool     `json:"isNew"`
	IsTop      bool     `json:"isTop"`
	Status     int8     `json:"status"`
	Sort       int      `json:"sort"`
	Views      int      `json:"views"`
	AddTime    string   `json:"addtime"`
}

// ArticleReq 文章新增/编辑入参（对齐前端 Article 提交字段）。
type ArticleReq struct {
	CategoryID uint   `json:"categoryId"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Content    string   `json:"content"`
	Cover      string   `json:"cover"`
	Tags       []string `json:"tags"` // 文章标签数组（存库前 join 成逗号分隔串）
	IsNew      bool     `json:"isNew"`
	IsTop      bool     `json:"isTop"`
	Status     *int8    `json:"status"` // 指针区分未传（默认1显示）
	Sort       int      `json:"sort"`
	Views      *int     `json:"views"` // 虚拟阅读量（指针区分未传：新增默认0、编辑不覆盖真实浏览量）
	AddTime    string   `json:"addtime"` // YYYY-MM-DD，空=今天
}

// CategoryReq 分类新增/编辑入参。
type CategoryReq struct {
	Name   string `json:"name"`
	EnName string `json:"enName"`
	Cover  string `json:"cover"`
	Sort   int    `json:"sort"`
}

func toCategoryView(c *model.ArticleCategory) CategoryView {
	return CategoryView{ID: c.ID, Name: c.Name, EnName: c.EnName, Cover: c.Cover, Sort: c.Sort}
}

func toArticleView(a *model.Article) ArticleView {
	return ArticleView{
		ID: a.ID, CategoryID: a.CategoryID, Title: a.Title, Summary: a.Summary,
		Content: a.Content, Cover: a.Cover, Tags: splitTags(a.Tags),
		IsNew: a.IsNew, IsTop: a.Top,
		Status: a.Active, Sort: a.Sort, Views: a.Count,
		AddTime: a.AddTime.Format(articleDateLayout),
	}
}

// splitTags 逗号分隔标签串 → 去空去重的数组（对外响应用）。
func splitTags(s string) []string {
	out := make([]string, 0, 4)
	seen := map[string]bool{}
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// joinTags 标签数组 → 去空去重的逗号分隔串（存库用）。
func joinTags(tags []string) string {
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return strings.Join(out, ",")
}

// ===== 分类 =====

func (s *ArticleService) ListCategories() ([]CategoryView, error) {
	list, err := s.repo.ListCategories()
	if err != nil {
		return nil, err
	}
	out := make([]CategoryView, 0, len(list))
	for i := range list {
		out = append(out, toCategoryView(&list[i]))
	}
	return out, nil
}

func (s *ArticleService) CreateCategory(req CategoryReq) (uint, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, artErr("分类名称不能为空")
	}
	c := &model.ArticleCategory{
		Name: name, EnName: strings.TrimSpace(req.EnName),
		Cover: strings.TrimSpace(req.Cover), Sort: req.Sort, AddTime: time.Now(),
	}
	if err := s.repo.CreateCategory(c); err != nil {
		return 0, err
	}
	return c.ID, nil
}

func (s *ArticleService) UpdateCategory(id uint, req CategoryReq) error {
	c, err := s.repo.FindCategory(id)
	if err != nil {
		return err
	}
	if c == nil {
		return artErr("分类不存在")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return artErr("分类名称不能为空")
	}
	return s.repo.UpdateCategory(id, map[string]interface{}{
		"name": name, "en_name": strings.TrimSpace(req.EnName),
		"cover": strings.TrimSpace(req.Cover), "sort": req.Sort,
	})
}

func (s *ArticleService) DeleteCategory(id uint) error {
	c, err := s.repo.FindCategory(id)
	if err != nil {
		return err
	}
	if c == nil {
		return artErr("分类不存在")
	}
	return s.repo.DeleteCategory(id)
}

// ===== 文章 =====

// List 后台文章列表（DB 分页 + 标题搜索，对齐 epay article.php）。
func (s *ArticleService) List(kw string, page, pageSize int) ([]ArticleView, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 30
	}
	list, total, err := s.repo.List(strings.TrimSpace(kw), false, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ArticleView, 0, len(list))
	for i := range list {
		out = append(out, toArticleView(&list[i]))
	}
	return out, total, nil
}

// PublicList 官网公开读取：全部显示中的文章（按分类分列，前端组装）。
func (s *ArticleService) PublicList() ([]ArticleView, error) {
	list, err := s.repo.ListAllActive()
	if err != nil {
		return nil, err
	}
	out := make([]ArticleView, 0, len(list))
	for i := range list {
		out = append(out, toArticleView(&list[i]))
	}
	return out, nil
}

// Detail 文章详情（公开）。incrView=true 时浏览量 +1（对齐 epay 查看 count+1）。
func (s *ArticleService) Detail(id uint, incrView bool) (*ArticleView, error) {
	a, err := s.repo.FindArticle(id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, artErr("文章不存在")
	}
	if incrView {
		_ = s.repo.IncrCount(id) // 浏览量递增失败不阻断详情返回
		a.Count++
	}
	v := toArticleView(a)
	return &v, nil
}

// Create 新增文章（对齐 epay add_submit：标题/内容非空 + 标题唯一）。
func (s *ArticleService) Create(req ArticleReq) (uint, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" || strings.TrimSpace(req.Content) == "" {
		return 0, artErr("文章标题和内容不能为空")
	}
	n, err := s.repo.CountByTitle(title, 0)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, artErr("文章标题已存在")
	}
	a := &model.Article{
		CategoryID: req.CategoryID, Title: title, Summary: strings.TrimSpace(req.Summary),
		Content: req.Content, Cover: strings.TrimSpace(req.Cover), Tags: joinTags(req.Tags),
		IsNew: req.IsNew, Top: req.IsTop, Sort: req.Sort,
		Active:  activeFromReq(req.Status),
		Count:   viewsFromReq(req.Views),
		AddTime: parseArticleDate(req.AddTime),
	}
	if err := s.repo.CreateArticle(a); err != nil {
		return 0, err
	}
	return a.ID, nil
}

// Update 编辑文章（对齐 epay edit_submit）。
func (s *ArticleService) Update(id uint, req ArticleReq) error {
	a, err := s.repo.FindArticle(id)
	if err != nil {
		return err
	}
	if a == nil {
		return artErr("文章不存在")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" || strings.TrimSpace(req.Content) == "" {
		return artErr("文章标题和内容不能为空")
	}
	n, err := s.repo.CountByTitle(title, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return artErr("文章标题已存在")
	}
	fields := map[string]interface{}{
		"category_id": req.CategoryID, "title": title, "summary": strings.TrimSpace(req.Summary),
		"content": req.Content, "cover": strings.TrimSpace(req.Cover), "tags": joinTags(req.Tags),
		"is_new": req.IsNew, "top": req.IsTop, "sort": req.Sort,
		"active":   activeFromReq(req.Status),
		"add_time": parseArticleDate(req.AddTime),
	}
	// 虚拟阅读量：仅当前端传了才覆盖，避免用陈旧表单值覆盖已累积的真实浏览量。
	if req.Views != nil {
		fields["count"] = viewsFromReq(req.Views)
	}
	return s.repo.UpdateArticle(id, fields)
}

func (s *ArticleService) Delete(id uint) error {
	a, err := s.repo.FindArticle(id)
	if err != nil {
		return err
	}
	if a == nil {
		return artErr("文章不存在")
	}
	return s.repo.DeleteArticle(id)
}

// SetActive 显示/隐藏切换（对齐 epay ajax.php setActive）。
func (s *ArticleService) SetActive(id uint, active int8) error {
	if active != 0 && active != 1 {
		return artErr("状态值不合法")
	}
	a, err := s.repo.FindArticle(id)
	if err != nil {
		return err
	}
	if a == nil {
		return artErr("文章不存在")
	}
	return s.repo.SetActive(id, active)
}

// activeFromReq 未传 status 默认 1（显示）；传了则按传入（非0即1）。
func activeFromReq(status *int8) int8 {
	if status == nil {
		return 1
	}
	if *status == 0 {
		return 0
	}
	return 1
}

// viewsFromReq 虚拟阅读量：未传或负数按 0 处理。
func viewsFromReq(views *int) int {
	if views == nil || *views < 0 {
		return 0
	}
	return *views
}

// parseArticleDate 解析 YYYY-MM-DD，空或非法回退今天。
func parseArticleDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now()
	}
	t, err := time.ParseInLocation(articleDateLayout, s, time.Local)
	if err != nil {
		return time.Now()
	}
	return t
}
