package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/pkg/money"
)

// MerchantInfo 商户信息（对齐 epay Merchant::info）。含余额、结算信息、今昨订单统计。
func (s *MapiService) MerchantInfo(m *model.Merchant) (map[string]string, error) {
	now := time.Now()
	today := dayStart(now)
	yesterday := today.AddDate(0, 0, -1)

	orderNum, _ := s.orders.CountPaidByMerchant(m.UID, time.Time{})
	orderToday, _ := s.orders.CountPaidByMerchant(m.UID, today)
	// 昨日订单数：今日之前 - 昨日之前，用两次范围差；简化用日订单数聚合。
	moneyToday, _ := s.orders.SumPaidMoneyByMerchant(m.UID, today, now.AddDate(0, 0, 1))
	moneyYesterday, _ := s.orders.SumPaidMoneyByMerchant(m.UID, yesterday, today)
	orderYesterdayCnt, _ := s.orders.CountPaidByMerchant(m.UID, yesterday)
	orderLastday := orderYesterdayCnt - orderToday
	if orderLastday < 0 {
		orderLastday = 0
	}

	out := map[string]string{
		"code":               "0",
		"pid":                uintToStr(m.UID),
		"status":             fmt.Sprintf("%d", m.Status),
		"pay_status":         fmt.Sprintf("%d", m.Pay),
		"settle_status":      fmt.Sprintf("%d", m.Settle),
		"money":              money.String(m.Money),
		"settle_account":     m.Account,
		"settle_name":        m.Username,
		"order_num":          fmt.Sprintf("%d", orderNum),
		"order_num_today":    fmt.Sprintf("%d", orderToday),
		"order_num_lastday":  fmt.Sprintf("%d", orderLastday),
		"order_money_today":  money.String(moneyToday),
		"order_money_lastday": money.String(moneyYesterday),
	}
	return pruneEmpty(out), nil
}

// MerchantOrders 商户订单列表（对齐 epay Merchant::orders）。可选 limit(≤50)/offset/status。
// 返回 {code, data:[...]}。data 里每单字段对齐 pay/query。
func (s *MapiService) MerchantOrders(m *model.Merchant, params map[string]string) (map[string]interface{}, error) {
	limit := 10
	if v, err := strconv.Atoi(strings.TrimSpace(params["limit"])); err == nil && v > 0 {
		limit = v
	}
	if limit > 50 {
		limit = 50
	}
	offset := 0
	if v, err := strconv.Atoi(strings.TrimSpace(params["offset"])); err == nil && v > 0 {
		offset = v
	}
	q := dto.OrderQuery{Page: offset/limit + 1, PageSize: limit}
	if st := strings.TrimSpace(params["status"]); st != "" {
		if v, err := strconv.Atoi(st); err == nil {
			vv := v
			q.Status = &vv
		}
	}
	uid := m.UID
	q.UID = &uid
	q.Normalize()

	list, _, err := s.orders.List(q)
	if err != nil {
		return nil, err
	}
	data := make([]map[string]string, 0, len(list))
	for i := range list {
		o := &list[i]
		row := map[string]string{
			"trade_no":     o.TradeNo,
			"out_trade_no": o.OutTradeNo,
			"api_trade_no": o.APITradeNo,
			"type":         o.TypeName,
			"pid":          uintToStr(o.UID),
			"addtime":      o.AddTime.Format(timeLayout),
			"name":         o.Name,
			"money":        money.String(o.Money),
			"param":        o.Param,
			"buyer":        o.Buyer,
			"clientip":     o.IP,
			"status":       fmt.Sprintf("%d", o.Status),
			"refundmoney":  money.String(o.RefundMoney),
		}
		if o.EndTime != nil {
			row["endtime"] = o.EndTime.Format(timeLayout)
		}
		data = append(data, pruneEmpty(row))
	}
	// data 列表不参与回包签名（epay 回包签名只对顶层标量），handler 单独处理。
	return map[string]interface{}{"code": 0, "data": data}, nil
}

// ===== 代付族（对齐 epay Transfer::submit/query/proof/balance）=====

// TransferSubmit 发起代付（复用 TransferService.CreateByMerchant，走费率+余额校验+扣款事务）。
// 真实渠道打款依赖外部凭证，DB 层资金流转如实执行（对齐 C3）。
func (s *MapiService) TransferSubmit(m *model.Merchant, params map[string]string) (map[string]string, error) {
	if s.transfer == nil {
		return nil, mapiErr("代付服务不可用")
	}
	req := dto.TransferCreateReq{
		BizNo:    strings.TrimSpace(params["out_biz_no"]),
		Type:     strings.TrimSpace(params["type"]),
		Account:  strings.TrimSpace(params["account"]),
		Username: strings.TrimSpace(params["name"]),
		Money:    strings.TrimSpace(params["money"]),
		Desc:     strings.TrimSpace(params["remark"]),
	}
	bizNo, err := s.transfer.CreateByMerchantSigned(m.UID, req)
	if err != nil {
		return nil, toMapiErr(err)
	}
	t, err := s.transfer.repo.FindByBizNo(bizNo)
	if err != nil || t == nil {
		return nil, mapiErr("代付单创建后查询失败")
	}
	return s.transferResult(t), nil
}

// TransferQuery 代付查询（对齐 epay Transfer::query）。入参 out_biz_no。
func (s *MapiService) TransferQuery(m *model.Merchant, params map[string]string) (map[string]string, error) {
	t, err := s.findMerchantTransfer(m.UID, params)
	if err != nil {
		return nil, err
	}
	return s.transferResult(t), nil
}

// TransferProof 代付凭证（对齐 epay Transfer::proof）。真实回单依赖渠道凭证，如实返回待凭证。
func (s *MapiService) TransferProof(m *model.Merchant, params map[string]string) (map[string]string, error) {
	t, err := s.findMerchantTransfer(m.UID, params)
	if err != nil {
		return nil, err
	}
	out := map[string]string{
		"code":       "0",
		"out_biz_no": t.BizNo,
		"proof":      "", // 真实回单凭证依赖渠道打款接口凭证，暂无
		"msg":        "回单凭证依赖真实渠道打款凭证，当前环境无法获取",
	}
	return out, nil
}

// TransferBalance 代付可用余额（对齐 epay Transfer::balance）。返回可用余额 + 代付费率。
func (s *MapiService) TransferBalance(m *model.Merchant) (map[string]string, error) {
	rate := s.cfg.Str("transfer_rate")
	if strings.TrimSpace(rate) == "" {
		rate = s.cfg.Str("settle_rate") // 空则复用结算费率（对齐 transfer 配置回退）
	}
	out := map[string]string{
		"code":            "0",
		"available_money": money.String(m.Money),
		"transfer_rate":   rate,
	}
	return out, nil
}

// findMerchantTransfer 按 out_biz_no 查本商户代付单。
func (s *MapiService) findMerchantTransfer(uid uint, params map[string]string) (*model.Transfer, error) {
	bizNo := strings.TrimSpace(params["out_biz_no"])
	if bizNo == "" {
		return nil, mapiErrCode(-4, "代付交易号不能为空")
	}
	t, err := s.transfer.repo.FindByBizNo(bizNo)
	if err != nil {
		return nil, err
	}
	if t == nil || t.UID != uid {
		return nil, mapiErr("代付记录不存在")
	}
	return t, nil
}

// transferResult 代付单 → 返回契约。
func (s *MapiService) transferResult(t *model.Transfer) map[string]string {
	out := map[string]string{
		"code":       "0",
		"out_biz_no": t.BizNo,
		"status":     fmt.Sprintf("%d", t.Status),
		"amount":     money.String(t.Money),
		"cost_money": money.String(t.CostMoney),
	}
	if t.PayTime != nil {
		out["paydate"] = t.PayTime.Format(timeLayout)
	}
	if t.Result != "" {
		out["msg"] = t.Result
	}
	return pruneEmpty(out)
}
