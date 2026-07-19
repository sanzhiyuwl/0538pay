package service

import (
	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/pkg/money"
)

const timeLayout = "2006-01-02 15:04:05" // 对齐前端 mock 的时间格式

// OrderService 订单业务逻辑。
type OrderService struct {
	repo *repository.OrderRepo
}

func NewOrderService(repo *repository.OrderRepo) *OrderService {
	return &OrderService{repo: repo}
}

// List 返回分页订单（已转为对外 View，金额补两位小数、时间格式化）。
func (s *OrderService) List(q dto.OrderQuery) ([]dto.OrderView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.OrderView, 0, len(list))
	for i := range list {
		views = append(views, toOrderView(&list[i]))
	}
	return views, total, nil
}

func toOrderView(o *model.Order) dto.OrderView {
	v := dto.OrderView{
		TradeNo:     o.TradeNo,
		OutTradeNo:  o.OutTradeNo,
		APITradeNo:  o.APITradeNo,
		UID:         o.UID,
		Domain:      o.Domain,
		Name:        o.Name,
		Money:       money.String(o.Money),
		GetMoney:    money.String(o.GetMoney),
		RefundMoney: money.String(o.RefundMoney),
		ProfitMoney: money.String(o.ProfitMoney),
		Type:        o.Type,
		TypeName:    o.TypeName,
		TypeShow:    o.TypeShow,
		Channel:     o.Channel,
		Plugin:      o.Plugin,
		IP:          o.IP,
		Buyer:       o.Buyer,
		AddTime:     o.AddTime.Format(timeLayout),
		Status:      o.Status,
		Settle:      o.Settle,
		Combine:     o.Combine,
	}
	if o.RealMoney != nil {
		s := money.String(*o.RealMoney)
		v.RealMoney = &s
	}
	if o.EndTime != nil {
		s := o.EndTime.Format(timeLayout)
		v.EndTime = &s
	}
	return v
}
