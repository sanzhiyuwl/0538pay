package service

import (
	"errors"
	"testing"
	"time"

	"github.com/epvia/api/internal/config"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
	"github.com/shopspring/decimal"
)

// TestAgentDeleteGuard 删代理守卫式物理删（连本地 MySQL）。
// 验证：① 纯净代理可删且连空钱包一并清（无孤儿）；② 有名额流水/进件单/邀请的代理一律拒删并提示改停用。
// 用高位测试 id，跑完清库，不污染业务数据。
func TestAgentDeleteGuard(t *testing.T) {
	db, err := model.NewDB(config.DatabaseConfig{
		DSN:     "pay0538:pay0538@tcp(127.0.0.1:3306)/pay0538?charset=utf8mb4&parseTime=True&loc=Local",
		MaxOpen: 4, MaxIdle: 2,
	})
	if err != nil {
		t.Skipf("跳过：连不上本地 MySQL（%v）", err)
	}
	agentRepo := repository.NewAgentRepo(db)
	enrollRepo := repository.NewEnrollRepo(db)
	svc := NewAgentService(agentRepo)
	svc.SetEnrollRepo(enrollRepo)

	// 用极高位 id 直插，避开自增真实代理；测试全程手动指定 id 便于清理。
	const (
		idClean   uint = 990001 // 纯净代理：可删
		idHasLog  uint = 990002 // 有名额流水：拒删
		idHasEnr  uint = 990003 // 有进件单：拒删
		idHasInv  uint = 990004 // 有邀请：拒删
	)
	ids := []uint{idClean, idHasLog, idHasEnr, idHasInv}

	cleanup := func() {
		for _, id := range ids {
			db.Where("agent_id = ?", id).Delete(&model.AgentQuotaLog{})
			db.Delete(&model.AgentQuotaWallet{}, "agent_id = ?", id)
			db.Where("agent_id = ?", id).Delete(&model.SubMerchantEnroll{})
			db.Where("agent_id = ?", id).Delete(&model.EnrollInvite{})
			db.Delete(&model.Agent{}, id)
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	// 建四个代理行（含随建空钱包，模拟 Create 行为）。
	for _, id := range ids {
		if err := db.Create(&model.Agent{
			ID: id, Name: "守卫测试", Account: "guard_" + itoa(id), Password: "x", Status: 1, AddTime: time.Now(),
		}).Error; err != nil {
			t.Fatalf("建测试代理 %d 失败: %v", id, err)
		}
		if err := db.Create(&model.AgentQuotaWallet{AgentID: id, UpdatedAt: time.Now()}).Error; err != nil {
			t.Fatalf("建测试钱包 %d 失败: %v", id, err)
		}
	}

	// ① 纯净代理：可删，且钱包一并清掉（无孤儿）。
	if err := svc.Delete(idClean); err != nil {
		t.Fatalf("纯净代理应可删，却失败: %v", err)
	}
	var agN, wN int64
	db.Model(&model.Agent{}).Where("id = ?", idClean).Count(&agN)
	db.Model(&model.AgentQuotaWallet{}).Where("agent_id = ?", idClean).Count(&wN)
	if agN != 0 || wN != 0 {
		t.Errorf("纯净代理删除后应无残留：agent=%d wallet=%d（期望 0/0）", agN, wN)
	}

	// ② 有名额流水：拒删。
	if err := agentRepo.ChangeQuota(idHasLog, "purchase", 1, decimal.NewFromInt(50), "", "测试售卖"); err != nil {
		t.Fatalf("造名额流水失败: %v", err)
	}
	assertBlocked(t, svc, idHasLog, "有名额流水")

	// ③ 有进件单：拒删。
	if err := enrollRepo.CreateEnroll(&model.SubMerchantEnroll{
		EnrollNo: "ENGUARD003", AgentID: idHasEnr, MerchantName: "守卫商户",
		Status: model.EnrollStatusPendingPay, AddTime: time.Now(),
	}); err != nil {
		t.Fatalf("造进件单失败: %v", err)
	}
	assertBlocked(t, svc, idHasEnr, "有进件单")

	// ④ 有邀请：拒删。
	if err := enrollRepo.CreateInvite(&model.EnrollInvite{
		Code: "INVGUARD004", AgentID: idHasInv, Status: model.InviteStatusEnabled, AddTime: time.Now(),
	}); err != nil {
		t.Fatalf("造邀请失败: %v", err)
	}
	assertBlocked(t, svc, idHasInv, "有邀请")
}

// assertBlocked 断言删除被守卫拦下（返回 AgentError），且代理行仍在。
func assertBlocked(t *testing.T, svc *AgentService, id uint, scene string) {
	t.Helper()
	err := svc.Delete(id)
	if err == nil {
		t.Fatalf("%s 的代理应被拒删，却成功了", scene)
	}
	var ae *AgentError
	if !errors.As(err, &ae) {
		t.Fatalf("%s 应返回 AgentError 业务提示，得到: %v", scene, err)
	}
}

// itoa 小工具：uint 转十进制串（避免引入 strconv 仅为拼账号）。
func itoa(u uint) string {
	if u == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for u > 0 {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
	}
	return string(b[i:])
}
