package service

import (
	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/pkg/money"
)

const timeLayout = "2006-01-02 15:04:05" // 对齐前端 mock 的时间格式

// OrderService 订单业务逻辑（含后台订单写操作）。
type OrderService struct {
	repo     *repository.OrderRepo
	accounts *repository.AccountRepo
	channels *repository.ChannelRepo
	admins   *repository.AdminRepo
	refunds  *repository.RefundOrderRepo // API 退款落 pay_refundorder（对齐 epay Order::refund api=1）
	pay      *PayService                 // 复用补单入账 + 重新通知
}

func NewOrderService(repo *repository.OrderRepo) *OrderService {
	return &OrderService{repo: repo}
}

// SetWriteDeps 注入订单写操作所需依赖（充值/通道/管理员/支付服务）。
// 用 setter 而非构造参数，避免打断既有 NewOrderService 调用点。
func (s *OrderService) SetWriteDeps(a *repository.AccountRepo, ch *repository.ChannelRepo, ad *repository.AdminRepo, pay *PayService) {
	s.accounts = a
	s.channels = ch
	s.admins = ad
	s.pay = pay
}

// SetRefundRepo 注入退款单仓储（API 退款写 pay_refundorder，对齐 epay Order::refund api=1 分支）。
func (s *OrderService) SetRefundRepo(r *repository.RefundOrderRepo) {
	s.refunds = r
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

// ExportRows 按当前筛选条件取全量订单（供后台流式 CSV 导出，不受列表分页 ≤100 限制）。
// 上限 100000 条对齐 epay download.php。返回对外 View。
func (s *OrderService) ExportRows(q dto.OrderQuery) ([]dto.OrderView, error) {
	list, err := s.repo.ExportAll(q, 100000)
	if err != nil {
		return nil, err
	}
	views := make([]dto.OrderView, 0, len(list))
	for i := range list {
		views = append(views, toOrderView(&list[i]))
	}
	return views, nil
}

// Stats 后台订单统计概况：全量聚合（同筛选），成功率 = (总-未付)/总*100 保留两位，对齐 epay。
func (s *OrderService) Stats(q dto.OrderQuery) (dto.OrderStats, error) {
	st, err := s.repo.Stats(q)
	if err != nil {
		return dto.OrderStats{}, err
	}
	if st.TotalCount > 0 {
		st.SuccessRate = float64(int(float64(st.TotalCount-st.UnpaidCount)/float64(st.TotalCount)*10000+0.5)) / 100
	}
	return st, nil
}

// ListByMerchant 商户端订单查询：强制限定当前商户 uid，防止越权查他人订单。
func (s *OrderService) ListByMerchant(uid uint, q dto.OrderQuery) ([]dto.OrderView, int64, error) {
	q.UID = &uid
	return s.List(q)
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
		BillTradeNo: o.BillTradeNo,
		Mobile:      o.Mobile,
		Param:       o.Param,
		Notify:      o.Notify,
	}
	if o.RealMoney != nil {
		s := money.String(*o.RealMoney)
		v.RealMoney = &s
	}
	if o.EndTime != nil {
		s := o.EndTime.Format(timeLayout)
		v.EndTime = &s
	}
	if o.RefundTime != nil {
		s := o.RefundTime.Format(timeLayout)
		v.RefundTime = &s
	}
	return v
}
