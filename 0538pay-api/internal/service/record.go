package service

import (
	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
)

// RecordService 资金流水业务逻辑。
type RecordService struct {
	repo *repository.RecordRepo
}

func NewRecordService(repo *repository.RecordRepo) *RecordService {
	return &RecordService{repo: repo}
}

// ListByMerchant 商户端资金流水查询：强制限定当前商户 uid。
func (s *RecordService) ListByMerchant(uid uint, q dto.RecordQuery) ([]dto.RecordView, int64, error) {
	q.Normalize()
	q.UID = &uid
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.RecordView, 0, len(list))
	for i := range list {
		views = append(views, toRecordView(&list[i]))
	}
	return views, total, nil
}

func toRecordView(r *model.PayRecord) dto.RecordView {
	return dto.RecordView{
		ID:       r.ID,
		Action:   r.Action,
		Money:    r.Money.InexactFloat64(),
		OldMoney: r.OldMoney.InexactFloat64(),
		NewMoney: r.NewMoney.InexactFloat64(),
		Type:     r.Type,
		TradeNo:  r.TradeNo,
		Date:     r.Date.Format(timeLayout),
	}
}
