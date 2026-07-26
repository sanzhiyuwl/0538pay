package service

import "log"

// aggOrWarn 收口只读聚合/展示查询的错误处理：这类查询失败通常不应阻断整个接口
// （对齐既有「失败退回零值、页面显示 0」的容错取舍），但必须留痕——否则一旦底层
// SQL 因改字段/列名拼写而崩掉（如曾发生的 realmoney→real_money），接口照返 200、
// 数字静默变 0，只能靠翻日志才能发现。此 helper 把「静默吞 error」统一改为
// 「记 warn 日志 + 返回零值」，既不改变原有不阻断的行为，又让故障可追溯。
//
// 用法：把 `v, _ := repo.SumXxx(...)` 改为 `v := aggOrWarn(repo.SumXxx(...))`
//（Go 多返回值展开要求被调函数是唯一实参，故不带业务标签；error 本身含 SQL/列名，
// 如 `Error 1054 Unknown column 'realmoney'`，足以定位是哪条查询崩了）。
// 仅用于只读聚合/展示查询；写操作与资金主链的 error 必须显式判断处理，不要用它吞掉。
func aggOrWarn[T any](v T, err error) T {
	if err != nil {
		log.Printf("[agg] 只读聚合查询失败(返回零值): %v", err)
	}
	return v
}
