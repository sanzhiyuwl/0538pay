package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/pkg/money"
)

// MerchantInfo 商户信息（对齐 epay Merchant::info）。含余额、结算信息、今昨订单统计。
func (s *MapiService) MerchantInfo(m *model.Merchant) (map[string]string, error) {
	now := time.Now()
	today := dayStart(now)
	yesterday := today.AddDate(0, 0, -1)

	// order_num 是全部订单数(不限状态)，对齐 epay Merchant::info count(*)（B1-14：原用已付计数偏小）。
	orderNum, _ := s.orders.CountAllByMerchant(m.UID)
	// 今日/昨日订单数按 [start,end) 范围精确统计（A-11，对齐 epay 按 date 精确，避免旧差值近似跨天出错）。
	orderToday, _ := s.orders.CountPaidByMerchantRange(m.UID, today, today.AddDate(0, 0, 1))
	orderLastday, _ := s.orders.CountPaidByMerchantRange(m.UID, yesterday, today)
	moneyToday, _ := s.orders.SumPaidMoneyByMerchant(m.UID, today, today.AddDate(0, 0, 1))
	moneyYesterday, _ := s.orders.SumPaidMoneyByMerchant(m.UID, yesterday, today)

	out := map[string]string{
		"code":               "0",
		"pid":                uintToStr(m.UID),
		"status":             fmt.Sprintf("%d", m.Status),
		"pay_status":         fmt.Sprintf("%d", m.Pay),
		"settle_status":      fmt.Sprintf("%d", m.Settle),
		"money":              money.String(m.Money),
		"settle_type":        fmt.Sprintf("%d", m.SettleID), // 结算方式(对齐 epay settle_type)
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
	var status *int
	if st := strings.TrimSpace(params["status"]); st != "" {
		if v, err := strconv.Atoi(st); err == nil {
			vv := v
			status = &vv
		}
	}
	// B1-15/16：对齐 epay Merchant::orders → ORDER BY trade_no DESC LIMIT {offset},{limit}，
	// 用原生行偏移(ListByMerchantV1)，不再把 offset 反算成 page（整除丢精度导致分页错位）。
	list, err := s.orders.ListByMerchantV1(m.UID, status, limit, offset)
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
		// B1-61：endtime 恒返回该键（epay 未支付单返回空串，非省略）。
		if o.EndTime != nil {
			row["endtime"] = o.EndTime.Format(timeLayout)
		} else {
			row["endtime"] = ""
		}
		// B1-17：epay orders() 无 array_filter，原样返回全字段，不删空字段。
		data = append(data, row)
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
	if s.transfer != nil { // B1-08：全局代付开关
		if err := s.transfer.CheckUserTransfer(m.GID); err != nil {
			return nil, toMapiErr(err)
		}
	}
	t, err := s.findMerchantTransfer(m.UID, params)
	if err != nil {
		return nil, err
	}
	return s.transferResult(t), nil
}

// TransferProof 代付凭证（对齐 epay Transfer::proof）。真实回单依赖渠道凭证，如实返回待凭证。
func (s *MapiService) TransferProof(m *model.Merchant, params map[string]string) (map[string]string, error) {
	if s.transfer != nil { // B1-08：全局代付开关
		if err := s.transfer.CheckUserTransfer(m.GID); err != nil {
			return nil, toMapiErr(err)
		}
	}
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
	if s.transfer != nil { // B1-08：全局代付开关
		if err := s.transfer.CheckUserTransfer(m.GID); err != nil {
			return nil, toMapiErr(err)
		}
	}
	rate := s.cfg.Str("transfer_rate")
	if strings.TrimSpace(rate) == "" {
		rate = s.cfg.Str("settle_rate") // 空则复用结算费率（对齐 transfer 配置回退）
	}
	// A-8：可用余额按 settle_type 计（settle_type=1 扣当日已收 realmoney），对齐 epay Transfer::balance。
	available := m.Money
	if s.transfer != nil {
		available = s.transfer.enableMoney(m)
	}
	out := map[string]string{
		"code":            "0",
		"available_money": money.String(available),
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

// transferResult 代付单 → 返回契约（B1-12，1:1 对齐 epay Transfer::query 三分支）。
// status=1 成功：msg='转账成功！', cost_money=costmoney, remark=desc。
// status=2 失败：msg='转账失败：原因', cost_money=money(注意是到账额非扣款额), errmsg=原因, remark=desc。
// 其它(处理中)：cost_money=costmoney, remark=desc。paydate/remark 恒返回该键（未支付为空串）。
func (s *MapiService) transferResult(t *model.Transfer) map[string]string {
	paydate := ""
	if t.PayTime != nil {
		paydate = t.PayTime.Format(timeLayout)
	}
	out := map[string]string{
		"code":       "0",
		"out_biz_no": t.BizNo,
		"status":     fmt.Sprintf("%d", t.Status),
		"amount":     money.String(t.Money),
		"paydate":    paydate,
		"remark":     t.Desc,
	}
	switch t.Status {
	case 1:
		out["msg"] = "转账成功！"
		out["cost_money"] = money.String(t.CostMoney)
	case 2:
		reason := t.Result
		if reason == "" {
			reason = "原因未知"
		}
		out["msg"] = "转账失败：" + reason
		out["errmsg"] = reason
		out["cost_money"] = money.String(t.Money) // 对齐 epay：失败分支 cost_money 取 money
	default:
		out["msg"] = "转账处理中"
		out["cost_money"] = money.String(t.CostMoney)
	}
	return out
}
