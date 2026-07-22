package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
)

// StatService 商户支付统计：对订单/代付实时聚合成交叉透视表（对齐 epay ustat/userPayStat）。
type StatService struct {
	stat     *repository.StatRepo
	channels *repository.ChannelRepo
}

func NewStatService(stat *repository.StatRepo, channels *repository.ChannelRepo) *StatService {
	return &StatService{stat: stat, channels: channels}
}

// 统计口径 type → 订单金额字段（type=4 代付走 transfer，单列）。
var statFieldByType = map[int]string{
	0: "money",         // 订单金额
	1: "real_money",    // 支付金额
	2: "get_money",     // 分成金额
	3: "profit_money",  // 手续费利润
}

// BuyerStat 支付用户统计（C-3，对齐 epay buyerStat）。按 method 选付款人维度，范围内聚合次数/金额 + 黑名单标记。
func (s *StatService) BuyerStat(q dto.BuyerStatQuery) ([]dto.BuyerStatRow, error) {
	column := "buyer"
	switch q.Method {
	case 1:
		column = "ip"
	case 2:
		column = "mobile"
	}
	start, end := statRange(q.StartDay, q.EndDay)
	rows, err := s.stat.BuyerStat(column, q.Type, start, end)
	if err != nil {
		return nil, err
	}
	out := make([]dto.BuyerStatRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.BuyerStatRow{
			User:    r.User,
			Count:   r.OrderCount,
			Amount:  r.Amount.StringFixed(2),
			IsBlack: r.IsBlack,
		})
	}
	return out, nil
}

// PayStat 按 method(type/channel) + 口径 type 聚合，产出交叉透视表。
func (s *StatService) PayStat(q dto.StatQuery) (*dto.StatResult, error) {
	byChannel := q.Method == "channel"
	start, end := statRange(q.StartDay, q.EndDay)

	// 列定义：按通道则列=已开启通道；按方式则列=出现过的支付方式名。
	columns, colLabel := s.buildColumns(byChannel)

	var cells []repository.StatCell
	var err error
	if q.Type == 4 {
		// 代付金额：聚合 pay_transfer
		cells, err = s.stat.AggregateTransfers(byChannel, start, end)
	} else {
		field := statFieldByType[q.Type]
		if field == "" {
			field = "money"
		}
		cells, err = s.stat.AggregateOrders(field, byChannel, start, end)
	}
	if err != nil {
		return nil, err
	}

	// 组装 uid → 列key → 金额
	rowMap := map[uint]map[string]decimal.Decimal{}
	for _, c := range cells {
		key := c.GroupBy
		if byChannel {
			key = "ch_" + c.GroupBy // 通道列 key 前缀
		}
		if _, ok := rowMap[c.UID]; !ok {
			rowMap[c.UID] = map[string]decimal.Decimal{}
		}
		rowMap[c.UID][key] = rowMap[c.UID][key].Add(c.Amount)
		// 动态补列（按方式时列来自实际数据）
		if !byChannel {
			if _, ok := colLabel[key]; !ok {
				colLabel[key] = c.GroupBy
				columns = append(columns, dto.StatColumn{Key: key, Label: c.GroupBy})
			}
		}
	}

	// 生成行 + 列合计
	rows := make([]dto.StatRow, 0, len(rowMap))
	totals := map[string]float64{}
	grand := decimal.Zero
	for uid, vals := range rowMap {
		values := map[string]float64{}
		rowTotal := decimal.Zero
		for _, col := range columns {
			amt := vals[col.Key]
			values[col.Key] = amt.InexactFloat64()
			rowTotal = rowTotal.Add(amt)
			totals[col.Key] += amt.InexactFloat64()
		}
		grand = grand.Add(rowTotal)
		rows = append(rows, dto.StatRow{
			UID:    uid,
			Name:   statMerchantName(uid),
			Values: values,
			Total:  rowTotal.InexactFloat64(),
		})
	}
	// 按合计降序（对齐 epay 展示习惯）
	sortStatRows(rows)

	return &dto.StatResult{
		Columns: columns,
		Rows:    rows,
		Totals:  totals,
		Grand:   grand.InexactFloat64(),
	}, nil
}

// buildColumns 生成列定义。按通道：列=已开启通道(ch_<id>)；按方式：初始空，聚合时动态补。
func (s *StatService) buildColumns(byChannel bool) ([]dto.StatColumn, map[string]string) {
	columns := []dto.StatColumn{}
	labels := map[string]string{}
	if byChannel {
		one := 1
		list, _, err := s.channels.List(dto.ChannelQuery{Page: 1, PageSize: 100, Status: &one})
		if err == nil {
			for i := range list {
				c := &list[i]
				key := "ch_" + strconv.FormatUint(uint64(c.ID), 10)
				columns = append(columns, dto.StatColumn{Key: key, Label: c.Name})
				labels[key] = c.Name
			}
		}
	}
	return columns, labels
}

// statRange 解析统计时间范围 [start, end)。空则默认当天。
func statRange(startDay, endDay string) (time.Time, time.Time) {
	loc := time.Local
	var start, end time.Time
	if t, err := time.ParseInLocation("2006-01-02", startDay, loc); err == nil {
		start = t
	} else {
		now := time.Now()
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	}
	if t, err := time.ParseInLocation("2006-01-02", endDay, loc); err == nil {
		end = t.AddDate(0, 0, 1) // 含结束日当天
	} else {
		end = start.AddDate(0, 0, 1)
	}
	return start, end
}

func statMerchantName(uid uint) string {
	if uid == 0 {
		return "管理员"
	}
	return fmt.Sprintf("商户%d", uid)
}

// sortStatRows 按 Total 降序排序（简单插入排序，行数不大）。
func sortStatRows(rows []dto.StatRow) {
	for i := 1; i < len(rows); i++ {
		j := i
		for j > 0 && rows[j].Total > rows[j-1].Total {
			rows[j], rows[j-1] = rows[j-1], rows[j]
			j--
		}
	}
}
