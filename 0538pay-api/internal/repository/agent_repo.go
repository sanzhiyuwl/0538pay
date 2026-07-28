package repository

import (
	"time"

	"github.com/epvia/api/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AgentRepo 代理数据访问：代理 CRUD + 名额钱包/流水（钱包变动走事务，不裸改）。
type AgentRepo struct{ db *gorm.DB }

func NewAgentRepo(db *gorm.DB) *AgentRepo { return &AgentRepo{db: db} }

// List 分页查询代理，支持 名称/账号 关键词、状态筛选。
func (r *AgentRepo) List(keyword string, status *int8, page, pageSize int) ([]model.Agent, int64, error) {
	tx := r.db.Model(&model.Agent{})
	if keyword != "" {
		tx = tx.Where("name LIKE ? OR account LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status != nil {
		tx = tx.Where("status = ?", *status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Agent
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// FindByID 按 id 取代理。
func (r *AgentRepo) FindByID(id uint) (*model.Agent, error) {
	var a model.Agent
	if err := r.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// FindByAccount 按登录账号取代理（登录用）。
func (r *AgentRepo) FindByAccount(account string) (*model.Agent, error) {
	var a model.Agent
	if err := r.db.Where("account = ?", account).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Create 新建代理，同时初始化空名额钱包（一代理一钱包）。
func (r *AgentRepo) Create(a *model.Agent) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		a.AddTime = time.Now()
		if err := tx.Create(a).Error; err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.AgentQuotaWallet{AgentID: a.ID, UpdatedAt: time.Now()}).Error
	})
}

// Update 更新代理可编辑字段（名称/联系方式/状态/权限/备注）。密码单独走 UpdatePassword。
func (r *AgentRepo) Update(id uint, fields map[string]any) error {
	return r.db.Model(&model.Agent{}).Where("id = ?", id).Updates(fields).Error
}

// UpdatePassword 单独更新密码（bcrypt 密文）。
func (r *AgentRepo) UpdatePassword(id uint, hash string) error {
	return r.db.Model(&model.Agent{}).Where("id = ?", id).Update("password", hash).Error
}

// Delete 物理删除代理（项目不使用软删除）。
func (r *AgentRepo) Delete(id uint) error {
	return r.db.Delete(&model.Agent{}, id).Error
}

// —— 名额钱包 ——

// Wallet 取代理名额钱包，无则返回零值钱包（不报错，便于展示）。
func (r *AgentRepo) Wallet(agentID uint) (*model.AgentQuotaWallet, error) {
	var w model.AgentQuotaWallet
	err := r.db.First(&w, "agent_id = ?", agentID).Error
	if err == gorm.ErrRecordNotFound {
		return &model.AgentQuotaWallet{AgentID: agentID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// ChangeQuota 名额变动的原子事务：行锁读钱包 → 校验 → 改余额 → 写流水。
// change>0 购买/退回，change<0 消耗。消耗时余额不足返回 ErrInsufficientBalance（复用结算域哨兵）。
func (r *AgentRepo) ChangeQuota(agentID uint, typ string, change int, amount decimal.Decimal, relNo, remark string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var w model.AgentQuotaWallet
		// 行锁读；钱包不存在则先建（代理创建时已建，这里兜底）。
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&w, "agent_id = ?", agentID).Error
		if err == gorm.ErrRecordNotFound {
			w = model.AgentQuotaWallet{AgentID: agentID}
			if e := tx.Create(&w).Error; e != nil {
				return e
			}
			err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&w, "agent_id = ?", agentID).Error
		}
		if err != nil {
			return err
		}
		before := w.Balance
		after := before + change
		if after < 0 {
			return ErrInsufficientBalance
		}
		fields := map[string]any{"balance": after, "updated_at": time.Now()}
		if change > 0 {
			fields["total_buy"] = w.TotalBuy + change
		} else if change < 0 {
			fields["total_used"] = w.TotalUsed - change // change 为负，减去=加绝对值
		}
		if err := tx.Model(&model.AgentQuotaWallet{}).Where("agent_id = ?", agentID).Updates(fields).Error; err != nil {
			return err
		}
		return tx.Create(&model.AgentQuotaLog{
			AgentID: agentID, Type: typ, Change: change, Before: before, After: after,
			Amount: amount, RelNo: relNo, AddTime: time.Now(), Remark: remark,
		}).Error
	})
}

// QuotaLogs 分页查询名额流水（可按代理过滤）。
func (r *AgentRepo) QuotaLogs(agentID *uint, page, pageSize int) ([]model.AgentQuotaLog, int64, error) {
	tx := r.db.Model(&model.AgentQuotaLog{})
	if agentID != nil {
		tx = tx.Where("agent_id = ?", *agentID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AgentQuotaLog
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}
