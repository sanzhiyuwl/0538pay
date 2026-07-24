package service

import (
	"strings"
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// AnnounceService 网站公告（对齐 epay admin/gonggao.php + pre_anounce）。
type AnnounceService struct {
	repo *repository.AnnounceRepo
}

func NewAnnounceService(repo *repository.AnnounceRepo) *AnnounceService {
	return &AnnounceService{repo: repo}
}

// List 后台全部公告。
func (s *AnnounceService) List() ([]dto.AdminAnnounceView, error) {
	list, err := s.repo.All()
	if err != nil {
		return nil, err
	}
	return toAnnounceViews(list), nil
}

// ListVisible 展示中的公告（官网/商户端读取）。
func (s *AnnounceService) ListVisible() ([]dto.AdminAnnounceView, error) {
	list, err := s.repo.ListVisible()
	if err != nil {
		return nil, err
	}
	return toAnnounceViews(list), nil
}

// Create 新增公告。
func (s *AnnounceService) Create(req dto.AnnounceSaveReq) error {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return maErr("公告内容不能为空")
	}
	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}
	return s.repo.Create(&model.Announce{
		Content: content, Color: strings.TrimSpace(req.Color),
		Sort: req.Sort, Status: status, AddTime: time.Now(),
	})
}

// Update 编辑公告。
func (s *AnnounceService) Update(id uint, req dto.AnnounceSaveReq) error {
	a, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if a == nil {
		return maErr("公告不存在")
	}
	fields := map[string]interface{}{
		"content": strings.TrimSpace(req.Content),
		"color":   strings.TrimSpace(req.Color),
		"sort":    req.Sort,
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	return s.repo.Update(id, fields)
}

// SetStatus 显隐切换（1显示 0隐藏）。
func (s *AnnounceService) SetStatus(id uint, status int8) error {
	if status != 0 && status != 1 {
		return maErr("状态值不合法")
	}
	return s.repo.Update(id, map[string]interface{}{"status": status})
}

// Delete 删除公告。
func (s *AnnounceService) Delete(id uint) error { return s.repo.Delete(id) }

func toAnnounceViews(list []model.Announce) []dto.AdminAnnounceView {
	views := make([]dto.AdminAnnounceView, 0, len(list))
	for i := range list {
		a := &list[i]
		views = append(views, dto.AdminAnnounceView{
			ID: a.ID, Content: a.Content, Color: a.Color,
			Sort: a.Sort, Status: a.Status, AddTime: a.AddTime.Format(timeLayout),
		})
	}
	return views
}
