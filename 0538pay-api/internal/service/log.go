package service

import (
	"crypto/rand"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
)

// LogService 登录日志：写入 + 只读列表（对齐 epay pre_log）。
type LogService struct {
	repo *repository.LogRepo
}

func NewLogService(repo *repository.LogRepo) *LogService { return &LogService{repo: repo} }

// Record 写入一条登录日志（登录入口调用）。uid=0 表示管理员。
func (s *LogService) Record(uid uint, logType, ip, city string) {
	_ = s.repo.Create(&model.LoginLog{
		UID: uid, Type: logType, IP: ip, City: city, Date: time.Now(),
	})
}

// List 登录日志列表（只读）。
func (s *LogService) List(q dto.LogQuery) ([]dto.LogView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q.Column, q.Value, q.Page, q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.LogView, 0, len(list))
	for i := range list {
		l := &list[i]
		views = append(views, dto.LogView{
			ID: l.ID, UID: l.UID, Type: l.Type, IP: l.IP, City: l.City,
			Date: l.Date.Format(timeLayout),
		})
	}
	return views, total, nil
}

// InviteService 邀请码：CRUD + 生成 + 注册校验/核销。
type InviteService struct {
	repo *repository.InviteRepo
}

func NewInviteService(repo *repository.InviteRepo) *InviteService { return &InviteService{repo: repo} }

// List 邀请码列表。
func (s *InviteService) List(q dto.InviteQuery) ([]dto.InviteView, int64, error) {
	q.Normalize()
	var status *int
	if q.Status != nil && *q.Status >= 0 {
		status = q.Status
	}
	list, total, err := s.repo.List(q.Keyword, status, q.Page, q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.InviteView, 0, len(list))
	for i := range list {
		c := &list[i]
		v := dto.InviteView{
			ID: c.ID, Code: c.Code, Status: c.Status,
			AddTime: c.AddTime.Format(timeLayout), UID: c.UID,
		}
		if c.UseTime != nil {
			t := c.UseTime.Format(timeLayout)
			v.UseTime = &t
		}
		views = append(views, v)
	}
	return views, total, nil
}

// Generate 批量生成邀请码（8 位随机码）。num 限 1~200。返回生成个数。
func (s *InviteService) Generate(num int) (int, error) {
	if num <= 0 {
		return 0, rkErr("生成个数需大于 0")
	}
	if num > 200 {
		num = 200
	}
	now := time.Now()
	codes := make([]model.InviteCode, 0, num)
	for i := 0; i < num; i++ {
		codes = append(codes, model.InviteCode{
			Code: randomCode(8), Status: 0, AddTime: now,
		})
	}
	if err := s.repo.BatchCreate(codes); err != nil {
		return 0, err
	}
	return num, nil
}

// Delete 删除单个邀请码。
func (s *InviteService) Delete(id uint) error { return s.repo.Delete(id) }

// ClearAll 清空全部。
func (s *InviteService) ClearAll() (int64, error) { return s.repo.ClearAll() }

// ClearUsed 清空已使用。
func (s *InviteService) ClearUsed() (int64, error) { return s.repo.ClearUsed() }

// Verify 校验邀请码可用（注册时）。返回可用记录 id，不可用返回 0。
func (s *InviteService) Verify(code string) (uint, bool) {
	c, err := s.repo.FindUsable(code)
	if err != nil || c == nil {
		return 0, false
	}
	return c.ID, true
}

// MarkUsed 注册成功核销邀请码。
func (s *InviteService) MarkUsed(id, uid uint) error {
	return s.repo.MarkUsed(id, uid, time.Now())
}

// inviteChars 邀请码字符集（大小写字母 + 数字，对齐 epay random(8)）。
const inviteChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// randomCode 生成 n 位随机邀请码（crypto/rand，避免可预测）。
func randomCode(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	out := make([]byte, n)
	for i := range b {
		out[i] = inviteChars[int(b[i])%len(inviteChars)]
	}
	return string(out)
}
