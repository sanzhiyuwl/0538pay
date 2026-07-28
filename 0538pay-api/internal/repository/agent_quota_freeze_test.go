package repository

import (
	"errors"
	"testing"

	"github.com/epvia/api/internal/config"
	"github.com/epvia/api/internal/model"
	"github.com/shopspring/decimal"
)

// TestQuotaFreezeLifecycle 名额三态（建单冻结 → 成功消耗 / 失败释放）全生命周期真跑（连本地 MySQL）。
// 验证：冻结扣可用加冻结、可用不足拦、消耗把冻结转 total_used、释放把冻结退回可用、
// 释放幂等（无冻结可退不报错也不刷高余额）、消耗无冻结可转报错（防重复消耗）。
// 用一个高位测试 agent_id，跑完清库，不污染业务数据。
func TestQuotaFreezeLifecycle(t *testing.T) {
	db, err := model.NewDB(config.DatabaseConfig{
		DSN:     "pay0538:pay0538@tcp(127.0.0.1:3306)/pay0538?charset=utf8mb4&parseTime=True&loc=Local",
		MaxOpen: 4, MaxIdle: 2,
	})
	if err != nil {
		t.Skipf("跳过：连不上本地 MySQL（%v）", err)
	}
	const aid uint = 999999 // 高位测试 id，避开真实代理
	// 起点干净：清掉可能的残留。
	db.Where("agent_id = ?", aid).Delete(&model.AgentQuotaLog{})
	db.Delete(&model.AgentQuotaWallet{}, "agent_id = ?", aid)
	t.Cleanup(func() {
		db.Where("agent_id = ?", aid).Delete(&model.AgentQuotaLog{})
		db.Delete(&model.AgentQuotaWallet{}, "agent_id = ?", aid)
	})

	r := NewAgentRepo(db)

	// 先售 2 名额（走 ChangeQuota purchase）。
	if err := r.ChangeQuota(aid, "purchase", 2, decimal.NewFromInt(100), "", "测试售卖"); err != nil {
		t.Fatalf("售卖名额失败: %v", err)
	}
	assertWallet(t, r, aid, 2, 0, 2, 0)

	// 冻结两次：balance 2→0，frozen 0→2。
	if err := r.FreezeQuota(aid, "ENA", "建单A冻结"); err != nil {
		t.Fatalf("冻结A失败: %v", err)
	}
	if err := r.FreezeQuota(aid, "ENB", "建单B冻结"); err != nil {
		t.Fatalf("冻结B失败: %v", err)
	}
	assertWallet(t, r, aid, 0, 2, 2, 0)

	// 可用不足：再冻结应返回 ErrInsufficientBalance，钱包不变。
	if err := r.FreezeQuota(aid, "ENC", "建单C冻结"); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("可用不足应拦 ErrInsufficientBalance，得到: %v", err)
	}
	assertWallet(t, r, aid, 0, 2, 2, 0)

	// 消耗 A（进件成功）：frozen 2→1，total_used 0→1，balance 不动。
	if err := r.ConsumeFrozen(aid, "ENA", "A进件成功消耗"); err != nil {
		t.Fatalf("消耗A失败: %v", err)
	}
	assertWallet(t, r, aid, 0, 1, 2, 1)

	// 释放 B（进件失败/退款）：frozen 1→0，balance 0→1。
	if err := r.ReleaseFrozen(aid, "ENB", "B退款释放"); err != nil {
		t.Fatalf("释放B失败: %v", err)
	}
	assertWallet(t, r, aid, 1, 0, 2, 1)

	// 释放幂等：无冻结可退，不报错也不把 balance 刷高。
	if err := r.ReleaseFrozen(aid, "ENB", "B重复释放应幂等"); err != nil {
		t.Fatalf("重复释放不应报错: %v", err)
	}
	assertWallet(t, r, aid, 1, 0, 2, 1)

	// 消耗无冻结可转：应返回 ErrInsufficientBalance（防重复消耗/漏冻结），钱包不变。
	if err := r.ConsumeFrozen(aid, "ENX", "无冻结消耗应拦"); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("无冻结消耗应拦 ErrInsufficientBalance，得到: %v", err)
	}
	assertWallet(t, r, aid, 1, 0, 2, 1)

	// 流水条数核对：purchase1 + freeze2 + consume1 + release1 = 5（幂等释放/失败消耗不写流水）。
	var n int64
	db.Model(&model.AgentQuotaLog{}).Where("agent_id = ?", aid).Count(&n)
	if n != 5 {
		t.Errorf("流水条数=%d 期望 5（purchase1+freeze2+consume1+release1）", n)
	}
}

func assertWallet(t *testing.T, r *AgentRepo, aid uint, balance, frozen, totalBuy, totalUsed int) {
	t.Helper()
	w, err := r.Wallet(aid)
	if err != nil {
		t.Fatalf("读钱包失败: %v", err)
	}
	if w.Balance != balance || w.Frozen != frozen || w.TotalBuy != totalBuy || w.TotalUsed != totalUsed {
		t.Fatalf("钱包 = {balance:%d frozen:%d buy:%d used:%d}，期望 {balance:%d frozen:%d buy:%d used:%d}",
			w.Balance, w.Frozen, w.TotalBuy, w.TotalUsed, balance, frozen, totalBuy, totalUsed)
	}
}
