package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// ChannelControlService 子商户管控（风控第二段）业务。
//
// 数据源：channel_enroll(status=approved) 的 sub_mchid（在本服务商名下已开通、可能被管控的商户全集）。
// 刷新：调微信「查询子商户管控情况」(4012803072) 按 sub_mchid 查被管控能力及原因 → 落 pay_channel_control 快照。
// 展示：admin/risk-controls 总览页读快照拼视图 + 概览统计。
// 硬锁（下一批）：收单/退款/提现/分账拦截读本表快照，不每次现查微信（避免 429 限频、阻塞下单）。
//
// ★ 批量刷新必须限速（微信 429 RATELIMIT_EXCEEDED）：串行 + 每次间隔节流。
type ChannelControlService struct {
	enrolls  *repository.ChannelEnrollRepo
	controls *repository.ChannelControlRepo
	channels *repository.ChannelRepo
	merchants *repository.MerchantRepo
	submch   *SubMerchantService
	flows    *repository.ChannelControlFlowRepo // 风控第三段管控流水（可空，SetFlowRepo 注入）
}

// SetFlowRepo 注入管控流水仓储（风控第三段：详情抽屉处置流水时间线用）。可选依赖，不改构造签名。
func (s *ChannelControlService) SetFlowRepo(r *repository.ChannelControlFlowRepo) { s.flows = r }

func NewChannelControlService(
	enrolls *repository.ChannelEnrollRepo,
	controls *repository.ChannelControlRepo,
	channels *repository.ChannelRepo,
	merchants *repository.MerchantRepo,
	submch *SubMerchantService,
) *ChannelControlService {
	return &ChannelControlService{enrolls: enrolls, controls: controls, channels: channels, merchants: merchants, submch: submch}
}

// ChannelControlError 携带业务提示。
type ChannelControlError struct{ Msg string }

func (e *ChannelControlError) Error() string { return e.Msg }

func ccErr(msg string) *ChannelControlError { return &ChannelControlError{Msg: msg} }

// —— 枚举中文映射（官方 4012803072；未知枚举回退原值，不臆造）——

var limitedFunctionText = map[string]string{
	"NO_TRANSACTION_AND_RECHARGE":     "关闭收单和充值",
	"NO_PAYMENT":                      "关闭付款",
	"NO_WITHDRAWAL":                   "关闭提现",
	"NO_REFUND":                       "关闭退款",
	"NO_TRANSACTION":                  "关闭收单",
	"NO_PROFIT_SHARING":               "关闭分账分出",
	"NO_PAYMENT_POINT_COMPLETE_ORDER": "关闭支付分服务结单",
}

var limitationReasonTypeText = map[string]string{
	"LICENSE_ABNORMAL":                    "经营证照异常",
	"NO_TRADE":                            "无交易",
	"SETTLE_ACCOUNT_ABNORMAL":             "结算信息异常",
	"RISK_ABNORMAL":                       "风险异常",
	"OTHER":                               "其他",
	"INSPECT_ABNORMAL":                    "巡检异常",
	"INVALID_REPRESENTATIVE_INFORMATION":  "法定代表人/负责人资料异常",
	"INVALID_BUSINESS_STATUS":             "经营状态异常",
	"INVALID_BUSINESS_LICENSE":            "经营证照资料异常",
	"INVALID_BENEFICIARY_INFORMATION":     "受益所有人资料异常",
}

var recoverWayText = map[string]string{
	"IRRECOVERABLE":                                  "不可恢复",
	"MODIFY_SUBJECT_INFORMATION":                     "修改主体资料",
	"MODIFY_SETTLE_ACCOUNT_INFORMATION":              "修改结算银行账户",
	"VERIFY_INACTIVE_MERCHANT_IDENTITY":              "核实商户身份",
	"SUBMIT_OFFLINE_BUSINESS_SCENARIO_INFORMATION":   "提交线下经营场景信息",
	"SUBMIT_INFORMATION_FOR_APPEAL":                  "提交相关信息申诉",
	"RESOLVE_TRANSACTION_DISPUTES":                   "解决交易纠纷",
	"MODIFY_ADMINISTRATOR_INFORMATION":               "修改超级管理员",
	"CALL_CUSTOMER_SERVICE_AT_95017":                 "拨打微信支付客服电话 95017",
	"UPDATE_BUSINESS_SCENARIO_INFORMATION":           "补充经营场景信息",
	"SUBMIT_CDD_INFORMATION":                         "填写尽调信息",
	"WAITING_FOR_PLATFORM_REVIEW":                    "等待平台审核",
	"SUBMIT_UBO_INFORMATION":                         "补充受益所有人信息",
	"SIGN_ANTI_FRAUD_PLEDGE_AND_VERIFY_FACE":         "签署反诈承诺书并刷脸核实身份",
	"CONTACT_APPROPRIATE_AUTHORITY_FOR_CONSULTATION": "联系有权机关咨询",
	"MODIFY_ABBREVIATION_INFORMATION":                "修改商户简称",
	"CONFIRM_BUSINESS_TYPE":                          "确认经营类型",
	"SUBMIT_ONLINE_BUSINESS_SCENARIO_INFORMATION":    "提交线上经营场景信息",
	"CONTACT_SERVICE_PROVIDER":                       "联系服务商处理",
}

func enumText(m map[string]string, code string) string {
	if code == "" {
		return ""
	}
	if t, ok := m[code]; ok {
		return t
	}
	return code // 未知枚举回退原值，不臆造
}

func channelControlStateText(state string) string {
	switch state {
	case model.ChannelControlControlled:
		return "被管控"
	case model.ChannelControlDelayed:
		return "延迟管控"
	default:
		return "正常"
	}
}

// channelFlowStateText 管控流水业务状态中文（B 机制 business_state；未知回退原值，不臆造）。
var channelFlowStateText = map[string]string{
	model.ChannelFlowStatePunishment:          "新增管控",
	model.ChannelFlowStateRecovery:            "管控解除",
	model.ChannelFlowStateDelayPunishment:     "即将被管控（延迟管控预告）",
	model.ChannelFlowStatePunishmentCancelled: "延迟管控已撤销",
}

// computeControlState 由微信应答推断本地管控态。
//   立即管控（或延迟管控已到点，limitation_date 有值）→ controlled
//   延迟管控未到点（DELAY 且 limitation_date 空）→ delayed
//   无任何被管控能力 → normal
func computeControlState(r *MchLimitationResp) string {
	if r == nil {
		return model.ChannelControlNormal
	}
	immediate, delayPending := false, false
	for _, rec := range r.RecoverySpecifications {
		if rec.LimitationActionType == "LIMIT_ACTION_TYPE_DELAY_CONTROL" && strings.TrimSpace(rec.LimitationDate) == "" {
			delayPending = true
		} else {
			immediate = true
		}
	}
	switch {
	case immediate:
		return model.ChannelControlControlled
	case r.Limited() && len(r.RecoverySpecifications) == 0:
		return model.ChannelControlControlled
	case delayPending:
		return model.ChannelControlDelayed
	case r.Limited():
		return model.ChannelControlControlled
	default:
		return model.ChannelControlNormal
	}
}

// List 风控总览：已开通子商户全集 + 各自快照 + 概览统计（admin，全量）。
func (s *ChannelControlService) List(ctx context.Context) (*dto.ChannelControlListResp, error) {
	enrolls, err := s.enrolls.ListApproved()
	if err != nil {
		return nil, err
	}
	return s.buildListResp(enrolls)
}

// ListForMerchant 商户端「业务受限」面板：只读，强制按登录商户 uid 隔离，只看自己名下已开通子商户。
// 与 admin List 共用 buildListResp（★两处不重复造轮子：同一份快照数据，仅数据源维度不同）。
func (s *ChannelControlService) ListForMerchant(ctx context.Context, uid uint) (*dto.ChannelControlListResp, error) {
	enrolls, err := s.enrolls.ListApprovedByUID(uid)
	if err != nil {
		return nil, err
	}
	return s.buildListResp(enrolls)
}

// buildListResp 由进件单集合 + 各自快照拼装列表应答 + 概览统计（admin/商户端共用）。
func (s *ChannelControlService) buildListResp(enrolls []model.ChannelEnroll) (*dto.ChannelControlListResp, error) {
	ids := make([]uint, 0, len(enrolls))
	for i := range enrolls {
		ids = append(ids, enrolls[i].ID)
	}
	snaps, err := s.controls.MapByEnrollIDs(ids)
	if err != nil {
		return nil, err
	}
	nameCache := map[int]string{}
	phoneCache := map[uint]string{}
	views := make([]dto.ChannelControlView, 0, len(enrolls))
	var ov dto.ChannelControlOverview
	ov.ApprovedTotal = int64(len(enrolls))
	for i := range enrolls {
		e := &enrolls[i]
		v := s.buildView(e, snaps[e.ID], nameCache, phoneCache)
		switch v.State {
		case model.ChannelControlControlled:
			ov.Controlled++
		case model.ChannelControlDelayed:
			ov.Delayed++
		default:
			ov.Normal++
		}
		if !v.Queried {
			ov.NeverQueried++
		}
		views = append(views, v)
	}
	return &dto.ChannelControlListResp{Overview: ov, List: views}, nil
}

// GetByEnrollID 单个进件单的管控快照视图（进件详情抽屉「业务受限」就地快照用；
// 未刷新过也返回 Queried=false 的视图，不报错——这是「两处不重复造轮子」的落点，
// 与风控总览页共用同一份快照数据，仅按 enroll_id 取一条而非全量列表）。
// enroll 未开通（非 approved 或无 sub_mchid）返回 nil, nil（详情页判空跳过展示）。
func (s *ChannelControlService) GetByEnrollID(enrollID uint) (*dto.ChannelControlView, error) {
	e, err := s.enrolls.FindByID(enrollID)
	if err != nil {
		return nil, nil
	}
	if e.Status != model.ChannelEnrollApproved || e.SubMchID == "" {
		return nil, nil
	}
	snap, err := s.controls.FindByEnrollID(enrollID)
	if err != nil {
		return nil, err
	}
	v := s.buildView(e, snap, map[int]string{}, map[uint]string{})
	return &v, nil
}

// GetByEnrollIDForMerchant 商户端就地快照：先归属校验（enroll_id 须属登录商户 uid）再委托 GetByEnrollID。
func (s *ChannelControlService) GetByEnrollIDForMerchant(enrollID, uid uint) (*dto.ChannelControlView, error) {
	e, err := s.enrolls.FindByID(enrollID)
	if err != nil || e.UID != uid {
		return nil, ccErr("进件单不存在或不在你名下")
	}
	return s.GetByEnrollID(enrollID)
}

// buildView 由进件单 + 快照拼装对外视图。
func (s *ChannelControlService) buildView(e *model.ChannelEnroll, snap *model.ChannelControl, nameCache map[int]string, phoneCache map[uint]string) dto.ChannelControlView {
	name, ok := nameCache[e.ChannelID]
	if !ok {
		if ch, _ := s.channels.FindByID(uint(e.ChannelID)); ch != nil {
			name = ch.Name
		}
		nameCache[e.ChannelID] = name
	}
	phone, ok := phoneCache[e.UID]
	if !ok {
		if s.merchants != nil {
			if m, _ := s.merchants.FindByUIDSafe(e.UID); m != nil {
				phone = m.Phone
			}
		}
		phoneCache[e.UID] = phone
	}
	v := dto.ChannelControlView{
		EnrollID:      e.ID,
		UID:           e.UID,
		MerchantName:  e.MerchantName,
		MerchantPhone: phone,
		ChannelID:     e.ChannelID,
		ChannelName:   name,
		SubMchID:      e.SubMchID,
		State:         model.ChannelControlNormal,
		StateText:     channelControlStateText(model.ChannelControlNormal),
	}
	if snap == nil {
		return v // 尚未刷新过：Queried=false，态按正常展示（不代表真正正常，UI 提示「未刷新」）
	}
	// ★ recordApply 可能先于管控刷新建出一条只带代办留痕、LastQueryAt 为空的快照壳——
	//   仍按「未刷新过」处理管控态，不要因快照记录存在就误判 Queried=true。
	if snap.LastQueryAt != nil && !snap.LastQueryAt.IsZero() {
		v.Queried = true
		v.State = snap.State
		v.StateText = channelControlStateText(snap.State)
		v.OtherLimitedFunctions = snap.OtherLimitedFunctions
		v.LastError = snap.LastError
		v.LastQueryAt = snap.LastQueryAt.Format(timeLayout)
	}
	if snap.LastSettleApplyNo != "" && snap.LastSettleApplyAt != nil {
		v.LastSettleApplyNo = snap.LastSettleApplyNo
		v.LastSettleApplyAt = snap.LastSettleApplyAt.Format(timeLayout)
	}
	if snap.LastSubjectApplyNo != "" && snap.LastSubjectApplyAt != nil {
		v.LastSubjectApplyNo = snap.LastSubjectApplyNo
		v.LastSubjectApplyAt = snap.LastSubjectApplyAt.Format(timeLayout)
	}
	if !v.Queried {
		return v
	}
	if snap.LimitedFunctions != "" {
		var fns []string
		if json.Unmarshal([]byte(snap.LimitedFunctions), &fns) == nil {
			v.LimitedFunctions = fns
			v.LimitedFunctionTexts = make([]string, len(fns))
			for i, f := range fns {
				v.LimitedFunctionTexts[i] = enumText(limitedFunctionText, f)
			}
		}
	}
	if snap.RecoverySpecifications != "" {
		var recs []MchLimitationRecovery
		if json.Unmarshal([]byte(snap.RecoverySpecifications), &recs) == nil {
			v.Recovery = make([]dto.ChannelControlRecovery, 0, len(recs))
			for _, r := range recs {
				v.Recovery = append(v.Recovery, dto.ChannelControlRecovery{
					LimitationCaseID:         r.LimitationCaseID,
					LimitationReasonType:     r.LimitationReasonType,
					LimitationReasonTypeText: enumText(limitationReasonTypeText, r.LimitationReasonType),
					LimitationReason:         r.LimitationReason,
					LimitationReasonDescribe: r.LimitationReasonDescribe,
					RelateLimitations:        r.RelateLimitations,
					OtherRelateLimitations:   r.OtherRelateLimitations,
					RecoverWay:               r.RecoverWay,
					RecoverWayText:           enumText(recoverWayText, r.RecoverWay),
					RecoverWayParam:          r.RecoverWayParam,
					RecoverHelpURL:           r.RecoverHelpURL,
					LimitationActionType:     r.LimitationActionType,
					LimitationStartDate:      r.LimitationStartDate,
					LimitationDate:           r.LimitationDate,
				})
			}
		}
	}
	return v
}

// refreshOne 查微信 + 落快照，返回刷新后的视图。查询失败也落 last_error 快照（保留旧管控数据）。
func (s *ChannelControlService) refreshOne(ctx context.Context, e *model.ChannelEnroll, nameCache map[int]string, phoneCache map[uint]string) (dto.ChannelControlView, error) {
	if s.submch == nil || !s.submch.Configured() {
		return dto.ChannelControlView{}, ccErr("微信服务商凭证未配置，无法查询管控情况")
	}
	resp, raw, err := s.submch.QuerySubMchLimitation(ctx, e.SubMchID)
	if err != nil {
		// 查询失败：更新 last_error，但保留既有管控字段（不误清为正常）。
		if snap, _ := s.controls.FindByEnrollID(e.ID); snap != nil {
			snap.LastError = err.Error()
			_ = s.controls.Upsert(snap)
		} else {
			_ = s.controls.Upsert(&model.ChannelControl{
				EnrollID: e.ID, UID: e.UID, ChannelID: e.ChannelID, SubMchID: e.SubMchID,
				State: model.ChannelControlNormal, LastError: err.Error(),
			})
		}
		return dto.ChannelControlView{}, err
	}
	fnsJSON, _ := json.Marshal(resp.LimitedFunctions)
	recsJSON, _ := json.Marshal(resp.RecoverySpecifications)
	snap := &model.ChannelControl{
		EnrollID:               e.ID,
		UID:                    e.UID,
		ChannelID:              e.ChannelID,
		SubMchID:               e.SubMchID,
		State:                  computeControlState(resp),
		LimitedFunctions:       string(fnsJSON),
		OtherLimitedFunctions:  resp.OtherLimitedFunctions,
		RecoverySpecifications: string(recsJSON),
		RawJSON:                string(raw),
		LastError:              "",
	}
	if err := s.controls.Upsert(snap); err != nil {
		return dto.ChannelControlView{}, err
	}
	return s.buildView(e, snap, nameCache, phoneCache), nil
}

// ListFlows 取某进件单的管控流水时间线（风控第三段；未注入流水仓储返回空列表）。
func (s *ChannelControlService) ListFlows(enrollID uint) (*dto.ChannelControlFlowResp, error) {
	resp := &dto.ChannelControlFlowResp{List: []dto.ChannelControlFlowItem{}}
	if s.flows == nil {
		return resp, nil
	}
	flows, err := s.flows.ListByEnrollID(enrollID)
	if err != nil {
		return nil, err
	}
	for i := range flows {
		f := &flows[i]
		item := dto.ChannelControlFlowItem{
			ID:                f.ID,
			Mechanism:         f.Mechanism,
			EventType:         f.EventType,
			Summary:           f.Summary,
			SubMchID:          f.SubMchID,
			RecordID:          f.RecordID,
			CompanyName:       f.CompanyName,
			PunishPlan:        f.PunishPlan,
			PunishTime:        f.PunishTime,
			PunishDescription: f.PunishDescription,
			RiskType:          f.RiskType,
			RiskDescription:   f.RiskDescription,
			BusinessCode:      f.BusinessCode,
			BusinessState:     f.BusinessState,
			BusinessStateText: enumText(channelFlowStateText, f.BusinessState),
			BusinessTime:      f.BusinessTime,
		}
		if !f.CreatedAt.IsZero() {
			item.CreatedAt = f.CreatedAt.Format(timeLayout)
		}
		resp.List = append(resp.List, item)
	}
	return resp, nil
}

// RefreshOne 单个商户刷新管控状态。
func (s *ChannelControlService) RefreshOne(ctx context.Context, enrollID uint) (*dto.ChannelControlRefreshResp, error) {
	e, err := s.enrolls.FindByID(enrollID)
	if err != nil {
		return nil, ccErr("进件单不存在")
	}
	if e.Status != model.ChannelEnrollApproved || e.SubMchID == "" {
		return nil, ccErr("该商户尚未开通子商户号，无管控状态可查")
	}
	nameCache := map[int]string{}
	phoneCache := map[uint]string{}
	v, err := s.refreshOne(ctx, e, nameCache, phoneCache)
	if err != nil {
		return &dto.ChannelControlRefreshResp{Refreshed: 0, Failed: 1}, err
	}
	return &dto.ChannelControlRefreshResp{Refreshed: 1, Failed: 0, Views: []dto.ChannelControlView{v}}, nil
}

// RefreshOneForMerchant 商户端刷新自己名下单个子商户的管控状态（归属校验后才委托 RefreshOne；
// 不开放批量——商户端只给「我自己这一家」的操作面，批量刷新是平台运维能力，仅 admin）。
func (s *ChannelControlService) RefreshOneForMerchant(ctx context.Context, enrollID, uid uint) (*dto.ChannelControlRefreshResp, error) {
	e, err := s.enrolls.FindByID(enrollID)
	if err != nil || e.UID != uid {
		return nil, ccErr("进件单不存在或不在你名下")
	}
	return s.RefreshOne(ctx, enrollID)
}

// controlRefreshThrottle 批量刷新时每笔间隔（限速，避免微信 429 RATELIMIT_EXCEEDED）。
const controlRefreshThrottle = 250 * time.Millisecond

// RefreshAll 批量刷新全部已开通商户的管控状态（串行 + 限速）。单笔失败不中断整体。
func (s *ChannelControlService) RefreshAll(ctx context.Context) (*dto.ChannelControlRefreshResp, error) {
	if s.submch == nil || !s.submch.Configured() {
		return nil, ccErr("微信服务商凭证未配置，无法查询管控情况")
	}
	enrolls, err := s.enrolls.ListApproved()
	if err != nil {
		return nil, err
	}
	nameCache := map[int]string{}
	phoneCache := map[uint]string{}
	resp := &dto.ChannelControlRefreshResp{}
	for i := range enrolls {
		if i > 0 {
			select {
			case <-ctx.Done():
				return resp, ctx.Err()
			case <-time.After(controlRefreshThrottle):
			}
		}
		v, err := s.refreshOne(ctx, &enrolls[i], nameCache, phoneCache)
		if err != nil {
			resp.Failed++
			continue
		}
		resp.Refreshed++
		resp.Views = append(resp.Views, v)
	}
	return resp, nil
}

// —— 解脱路径主动代办（0804 方案第五部分补遗：recover_way=修改主体资料/修改结算账户时，
//    服务商可直接调 API 为商户发起变更，比只引导商户去「微信支付商家助手」小程序自助处理更高效）——

// mustEnrolled 取进件单并校验已开通（approved + sub_mchid 非空），代办前置校验共用。
func (s *ChannelControlService) mustEnrolled(enrollID uint) (*model.ChannelEnroll, error) {
	e, err := s.enrolls.FindByID(enrollID)
	if err != nil {
		return nil, ccErr("进件单不存在")
	}
	if e.Status != model.ChannelEnrollApproved || e.SubMchID == "" {
		return nil, ccErr("该商户尚未开通子商户号，无法代办变更")
	}
	return e, nil
}

// ModifySettlementFor 代该商户修改结算银行账户（recover_way=MODIFY_SETTLE_ACCOUNT_INFORMATION）。
// 复用 SubMerchantService.ModifySettlement（进件成功后售后接口，与本次风控页操作项共用同一实现）。
func (s *ChannelControlService) ModifySettlementFor(ctx context.Context, enrollID uint, req ModifySettlementReq) (string, error) {
	if s.submch == nil || !s.submch.Configured() {
		return "", ccErr("微信服务商凭证未配置，无法代办变更")
	}
	e, err := s.mustEnrolled(enrollID)
	if err != nil {
		return "", err
	}
	appNo, _, err := s.submch.ModifySettlement(ctx, e.SubMchID, req)
	if err != nil {
		return "", ccErr(err.Error())
	}
	_ = s.controls.RecordSettleApply(e.ID, e.UID, e.ChannelID, e.SubMchID, appNo) // 留痕失败不影响本次代办已提交成功
	return appNo, nil
}

// ModifySubjectInfoFor 代该商户提交主体资料变更申请（recover_way=MODIFY_SUBJECT_INFORMATION）。
// outRequestNo 由调用方生成（业务申请编号，幂等键；驳回重提传相同值覆盖原单）。
func (s *ChannelControlService) ModifySubjectInfoFor(ctx context.Context, enrollID uint, outRequestNo string, req dto.SubjectAlterReq) (string, error) {
	if s.submch == nil || !s.submch.Configured() {
		return "", ccErr("微信服务商凭证未配置，无法代办变更")
	}
	e, err := s.mustEnrolled(enrollID)
	if err != nil {
		return "", err
	}
	applyID, _, err := s.submch.ModifySubjectInfo(ctx, e.SubMchID, outRequestNo, req)
	if err != nil {
		return "", ccErr(err.Error())
	}
	_ = s.controls.RecordSubjectApply(e.ID, e.UID, e.ChannelID, e.SubMchID, applyID) // 留痕失败不影响本次代办已提交成功
	return applyID, nil
}
