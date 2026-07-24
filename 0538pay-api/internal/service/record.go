package service

import (
	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// RecordService 资金流水业务逻辑。
type RecordService struct {
	repo *repository.RecordRepo
}

func NewRecordService(repo *repository.RecordRepo) *RecordService {
	return &RecordService{repo: repo}
}

// ListByMerchant 商户端资金流水查询：强制限定当前商户 uid（覆盖入参，防越权）。
func (s *RecordService) ListByMerchant(uid uint, q dto.RecordQuery) ([]dto.RecordView, int64, error) {
	q.Normalize()
	q.UID = &uid
	return s.list(q)
}

// List 后台资金流水查询：按入参筛选（可指定 uid，也可全量），不强制注入商户号。
func (s *RecordService) List(q dto.RecordQuery) ([]dto.RecordView, int64, error) {
	q.Normalize()
	return s.list(q)
}

// list 共用的查询+装配。
func (s *RecordService) list(q dto.RecordQuery) ([]dto.RecordView, int64, error) {
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

// Stats 后台资金明细统计（当前筛选条件下的增/减金额与笔数）。
func (s *RecordService) Stats(q dto.RecordQuery) (dto.RecordStats, error) {
	q.Normalize()
	inc, dec, incCnt, decCnt, err := s.repo.Stats(q)
	if err != nil {
		return dto.RecordStats{}, err
	}
	return dto.RecordStats{
		IncMoney:   inc.InexactFloat64(),
		DecMoney:   dec.InexactFloat64(),
		TotalMoney: inc.Sub(dec).InexactFloat64(),
		IncCount:   incCnt,
		DecCount:   decCnt,
		TotalCount: incCnt + decCnt,
	}, nil
}

func toRecordView(r *model.PayRecord) dto.RecordView {
	return dto.RecordView{
		ID:       r.ID,
		UID:      r.UID,
		Action:   r.Action,
		Money:    r.Money.InexactFloat64(),
		OldMoney: r.OldMoney.InexactFloat64(),
		NewMoney: r.NewMoney.InexactFloat64(),
		Type:     r.Type,
		TradeNo:  r.TradeNo,
		Date:     r.Date.Format(timeLayout),
	}
}
