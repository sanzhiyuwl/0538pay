package service

import (
	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/pkg/money"
)

// 用户组名映射（对齐前端 mock/merchants.ts 的 groups）。
// 后续接入用户组表后改为 JOIN 派生。
var groupNames = map[int]string{
	0: "默认用户组",
	1: "普通商户",
	2: "VIP商户",
	3: "企业商户",
}

func groupName(gid int) string {
	if n, ok := groupNames[gid]; ok {
		return n
	}
	return "默认用户组"
}

// MerchantService 商户业务逻辑。
type MerchantService struct {
	repo *repository.MerchantRepo
}

func NewMerchantService(repo *repository.MerchantRepo) *MerchantService {
	return &MerchantService{repo: repo}
}

// List 返回分页商户（转对外 View：金额补两位小数、时间格式化、派生组名）。
func (s *MerchantService) List(q dto.MerchantQuery) ([]dto.MerchantView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.MerchantView, 0, len(list))
	for i := range list {
		views = append(views, toMerchantView(&list[i]))
	}
	return views, total, nil
}

func toMerchantView(m *model.Merchant) dto.MerchantView {
	v := dto.MerchantView{
		UID:       m.UID,
		GID:       m.GID,
		GroupName: groupName(m.GID),
		Money:     money.String(m.Money),
		SettleID:  m.SettleID,
		Account:   m.Account,
		Username:  m.Username,
		QQ:        m.QQ,
		Phone:     m.Phone,
		Email:     m.Email,
		URL:       m.URL,
		AddTime:   m.AddTime.Format(timeLayout),
		Status:    m.Status,
		Cert:      m.Cert,
		Pay:       m.Pay,
		Settle:    m.Settle,
		UpID:      m.UpID,
		Mode:      m.Mode,
		Deposit:   money.String(m.Deposit),
	}
	if m.GroupEnd != nil {
		s := m.GroupEnd.Format(timeLayout)
		v.EndTime = &s
	}
	return v
}
