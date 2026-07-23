package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/pkg/money"
)

// ApiV1Service 实现 epay 的 V1 遗留接口 api.php?act=（GET+明文key鉴权，code=1 语义，A-5）。
// 已被 V2 /api/mapi/* REST(code=0) 取代，但为兼容老商户对接而复刻：query/order/orders/settle/refund。
// refund 复用 MapiService 的 V2 退款核心（同 reducemoney/幂等语义），仅入参鉴权换成 V1 明文 key。
type ApiV1Service struct {
	merchants *repository.MerchantRepo
	orders    *repository.OrderRepo
	settles   *repository.SettleRepo
	cfg       *ConfigService
	mapi      *MapiService // 复用 V2 退款核心
}

func NewApiV1Service(m *repository.MerchantRepo, o *repository.OrderRepo, st *repository.SettleRepo,
	cfg *ConfigService, mapi *MapiService) *ApiV1Service {
	return &ApiV1Service{merchants: m, orders: o, settles: st, cfg: cfg, mapi: mapi}
}

// v1Fail 生成 V1 错误响应（code<0）。
func v1Fail(code int, msg string) map[string]interface{} {
	return map[string]interface{}{"code": code, "msg": msg}
}

// authV1 校验 V1 明文 key 鉴权（对齐 epay：key===userrow.key，keytype==1 拒绝）。
func (s *ApiV1Service) authV1(pidStr, key string) (*model.Merchant, map[string]interface{}) {
	pid := parseUint(pidStr)
	if pid == 0 {
		return nil, v1Fail(-3, "商户ID不存在")
	}
	m, err := s.merchants.FindByUIDSafe(pid)
	if err != nil || m == nil {
		return nil, v1Fail(-3, "商户ID不存在")
	}
	if key == "" || key != m.AppKey {
		return nil, v1Fail(-3, "商户密钥错误")
	}
	if m.KeyType == 1 {
		return nil, v1Fail(-3, "该商户只能使用RSA签名类型")
	}
	return m, nil
}

// Query act=query：商户信息 + 今昨订单数（对齐 epay api.php query，code=1）。
func (s *ApiV1Service) Query(q map[string]string) map[string]interface{} {
	m, fail := s.authV1(q["pid"], q["key"])
	if fail != nil {
		return fail
	}
	// orders 是全部订单数(不限状态)，对齐 epay api.php query count(*)（B1-57：原用已付计数偏小）。
	orders, _ := s.orders.CountAllByMerchant(m.UID)
	// 今/昨按 status=1 精确按日统计
	today := dayStart(time.Now())
	yesterday := today.AddDate(0, 0, -1)
	todayCnt, _ := s.orders.CountPaidByMerchantRange(m.UID, today, today.AddDate(0, 0, 1))
	lastdayCnt, _ := s.orders.CountPaidByMerchantRange(m.UID, yesterday, today)
	return map[string]interface{}{
		"code": 1, "pid": m.UID, "key": m.AppKey, "active": m.Status,
		"money": money.String(m.Money), "type": m.SettleID,
		"account": m.Account, "username": m.Username,
		"orders": orders, "orders_today": todayCnt, "orders_lastday": lastdayCnt,
	}
}

// Order act=order：单笔查单。支持 SYS_KEY 内部签名(sign+trade_no)或明文 key(pid+key)（对齐 epay）。
func (s *ApiV1Service) Order(q map[string]string) map[string]interface{} {
	var o *model.Order
	var err error
	if q["sign"] != "" && q["trade_no"] != "" {
		// 内部签名查单：md5(SYS_KEY.trade_no.SYS_KEY)
		syskey := s.cfg.Str("syskey")
		want := md5Hex(syskey + q["trade_no"] + syskey)
		// B1-62：严格大小写敏感全等，对齐 epay `md5(...) !== $_GET['sign']`（大写 hex 应被拒）。
		if syskey == "" || q["sign"] != want {
			return v1Fail(-3, "verify sign failed")
		}
		o, err = s.orders.FindByTradeNo(q["trade_no"])
	} else {
		m, fail := s.authV1(q["pid"], q["key"])
		if fail != nil {
			return fail
		}
		if tn := q["trade_no"]; tn != "" {
			o, err = s.orders.FindByTradeNoAndUID(tn, m.UID)
		} else if otn := q["out_trade_no"]; otn != "" {
			o, err = s.orders.FindByOut(m.UID, otn)
		} else {
			return v1Fail(-4, "订单号不能为空")
		}
	}
	if err != nil || o == nil {
		return v1Fail(-1, "订单号不存在")
	}
	res := map[string]interface{}{
		"code": 1, "msg": "succ",
		"trade_no": o.TradeNo, "out_trade_no": o.OutTradeNo,
		"api_trade_no": o.APITradeNo, "bill_trade_no": o.BillTradeNo,
		"type": o.TypeName, "pid": o.UID, "addtime": o.AddTime.Format(timeLayout),
		"name": o.Name, "money": money.String(o.Money), "param": o.Param,
		"buyer": o.Buyer, "status": o.Status, "payurl": o.QRCode,
	}
	// B1-61：endtime 恒返回该键（epay act=order 未支付单返回空串，非省略）。
	if o.EndTime != nil {
		res["endtime"] = o.EndTime.Format(timeLayout)
	} else {
		res["endtime"] = ""
	}
	return res
}

// Orders act=orders：订单列表（limit≤50/offset/status，对齐 epay，code=1+count+data）。
func (s *ApiV1Service) Orders(q map[string]string) map[string]interface{} {
	m, fail := s.authV1(q["pid"], q["key"])
	if fail != nil {
		return fail
	}
	limit := clampLimit(parseIntDef(q["limit"], 10))
	offset := parseIntDef(q["offset"], 0)
	var status *int
	if v, ok := q["status"]; ok && v != "" {
		n := parseIntDef(v, 0)
		status = &n
	}
	list, err := s.orders.ListByMerchantV1(m.UID, status, limit, offset)
	if err != nil {
		return v1Fail(-1, "查询订单记录失败！")
	}
	data := make([]map[string]interface{}, 0, len(list))
	for i := range list {
		o := &list[i]
		row := map[string]interface{}{
			"trade_no": o.TradeNo, "out_trade_no": o.OutTradeNo, "type": o.TypeName,
			"pid": o.UID, "addtime": o.AddTime.Format(timeLayout), "name": o.Name,
			"money": money.String(o.Money), "param": o.Param, "buyer": o.Buyer, "status": o.Status,
		}
		// B1-61：endtime 恒返回该键（epay 未支付单返回空串，非省略）。
		if o.EndTime != nil {
			row["endtime"] = o.EndTime.Format(timeLayout)
		} else {
			row["endtime"] = ""
		}
		data = append(data, row)
	}
	return map[string]interface{}{"code": 1, "msg": "查询订单记录成功！", "count": len(data), "data": data}
}

// Settle act=settle：结算记录（limit≤50/offset，对齐 epay，code=1+data）。
func (s *ApiV1Service) Settle(q map[string]string) map[string]interface{} {
	m, fail := s.authV1(q["pid"], q["key"])
	if fail != nil {
		return fail
	}
	limit := clampLimit(parseIntDef(q["limit"], 10))
	offset := parseIntDef(q["offset"], 0)
	// B1-59：原生行偏移 + id DESC，对齐 epay `order by id desc limit {offset},{limit}`。
	list, err := s.settles.ListByMerchantOffset(m.UID, limit, offset)
	if err != nil {
		return v1Fail(-1, "查询结算记录失败！")
	}
	data := make([]map[string]interface{}, 0, len(list))
	for i := range list {
		st := &list[i]
		// B1-58：epay `SELECT *` 全字段原样回吐，此处回吐 pay_settle 全部有意义字段（非仅6项）。
		row := map[string]interface{}{
			"id": st.ID, "uid": st.UID, "batch": st.Batch, "auto": st.Auto,
			"type": st.Type, "account": st.Account, "username": st.Username,
			"money": money.String(st.Money), "realmoney": money.String(st.RealMoney),
			"status": st.Status, "result": st.Result,
			"addtime": st.AddTime.Format(timeLayout),
		}
		if st.EndTime != nil {
			row["endtime"] = st.EndTime.Format(timeLayout)
		} else {
			row["endtime"] = ""
		}
		data = append(data, row)
	}
	return map[string]interface{}{"code": 1, "msg": "查询结算记录成功！", "data": data}
}

// Refund act=refund：退款（POST，明文 key 鉴权 + per-merchant refund 开关，复用 V2 退款核心）。
// 返回 V1 语义（code=1 成功）；V2 核心返回 code=0，此处映射。
func (s *ApiV1Service) Refund(q map[string]string) map[string]interface{} {
	// B1-60(a)：对齐 epay api.php:118 —— 先查全局 user_refund 开关(坏 pid 也先报此错)，再做 pid/key 鉴权。
	if !s.cfg.Bool("user_refund") {
		return v1Fail(-4, "未开启商户后台自助退款")
	}
	m, fail := s.authV1(q["pid"], q["key"])
	if fail != nil {
		return fail
	}
	if m.Refund == 0 {
		return v1Fail(-2, "商户未开启订单退款API接口")
	}
	// 复用 V2 退款核心（金额校验/reducemoney/幂等/写 refundorder）。
	out, err := s.mapi.PayRefund(m, q)
	if err != nil {
		return v1Fail(-1, err.Error())
	}
	// V2 out(code=0) → V1(code=1)
	res := map[string]interface{}{"code": 1}
	for k, v := range out {
		if k == "code" {
			continue
		}
		res[k] = v
	}
	return res
}

// ---- 小工具 ----

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func clampLimit(n int) int {
	if n <= 0 {
		return 10
	}
	if n > 50 {
		return 50
	}
	return n
}

func parseIntDef(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return def
	}
	return n
}
