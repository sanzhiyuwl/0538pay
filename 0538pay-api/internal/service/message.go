package service

import (
	"strings"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
)

// MessageService 站内信（我方新增功能，epay 无此实体）。
// 管理员下发（定向单个商户 / UID=0 全体广播），商户端收件箱查看 + 标记已读。
type MessageService struct {
	repo *repository.MessageRepo
}

func NewMessageService(repo *repository.MessageRepo) *MessageService {
	return &MessageService{repo: repo}
}

// Send 管理员下发站内信。UID=0 表示全体广播。
func (s *MessageService) Send(req dto.MessageSendReq) error {
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" {
		return maErr("标题不能为空")
	}
	if content == "" {
		return maErr("内容不能为空")
	}
	return s.repo.Create(&model.Message{
		UID:     req.UID,
		Title:   title,
		Content: content,
	})
}

// AdminList 后台分页列出所有已下发的站内信。
func (s *MessageService) AdminList(page, pageSize int) ([]dto.MessageView, int64, error) {
	page, pageSize = clampPage(page, pageSize)
	list, total, err := s.repo.AdminList(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.MessageView, 0, len(list))
	for i := range list {
		views = append(views, toMessageView(&list[i], list[i].IsRead == 1))
	}
	return views, total, nil
}

// AdminDelete 后台删除站内信。
func (s *MessageService) AdminDelete(id uint) error { return s.repo.Delete(id) }

// MerchantList 商户收件箱（定向 + 广播），带已读态。
func (s *MessageService) MerchantList(uid uint, page, pageSize int) ([]dto.MessageView, int64, error) {
	page, pageSize = clampPage(page, pageSize)
	list, total, err := s.repo.MerchantList(uid, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	reads, err := s.repo.ReadUIDs(uid)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.MessageView, 0, len(list))
	for i := range list {
		m := &list[i]
		read := m.IsRead == 1
		if m.UID == 0 { // 广播信已读态看回执表
			read = reads[m.ID]
		}
		views = append(views, toMessageView(m, read))
	}
	return views, total, nil
}

// MerchantRead 商户标记已读（限本人收到的信）。
func (s *MessageService) MerchantRead(uid, msgID uint) error {
	return s.repo.MarkRead(uid, msgID)
}

// UnreadCount 商户未读数（顶栏红点）。
func (s *MessageService) UnreadCount(uid uint) (int64, error) {
	return s.repo.UnreadCount(uid)
}

func toMessageView(m *model.Message, read bool) dto.MessageView {
	return dto.MessageView{
		ID:      m.ID,
		UID:     m.UID,
		Title:   m.Title,
		Content: m.Content,
		IsRead:  read,
		Date:    m.CreatedAt.Format(timeLayout),
	}
}

// clampPage 归一化分页参数（复用给站内信/邀请等列表）。
func clampPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
