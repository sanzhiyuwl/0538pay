package service

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// ChannelEnrollService 服务商通道商户进件业务（epay 精仿线，只走商户进件不走二清，0730 阶段1）。
//
// 阶段1 半自动闭环：
//   商户端 /m 选服务商主通道建单(draft) → 填全套资料(敏感字段 RSA 加密落库) → 提交待审(pending)
//   → 平台后台人工审核 → 通过(approved)：系统自动建/更新该商户该通道的子通道，把上游子商户号写进
//     占位符括号内 key（我方微信服务商渠道 config 键=sub_mchid、富友=appmchid）、置子通道 status=1、
//     回填本单 SubChannelID(=epay apply_id)
//   → 或驳回(rejected)：记原因，商户可改料重提（复用同一单）。
//
// 敏感字段（证件号/银行账号/开户名）用平台 RSA 公钥(sys_rsa_public)加密落 material_json，
// 后台审核时用私钥(sys_rsa_private)解密报送上游；列表/回显只给脱敏 has_* 与掩码。
type ChannelEnrollService struct {
	repo      *repository.ChannelEnrollRepo
	channels  *repository.ChannelRepo
	subs      *repository.SubChannelRepo
	merchants *repository.MerchantRepo // 派生归属商户注册手机（后台列表展示）
	cfg       *ConfigService
	submch    *SubMerchantService // 微信服务商 applyment4sub 引擎（提交/上传/查状态），与代理线共用
}

func NewChannelEnrollService(
	repo *repository.ChannelEnrollRepo,
	channels *repository.ChannelRepo,
	subs *repository.SubChannelRepo,
	merchants *repository.MerchantRepo,
	cfg *ConfigService,
) *ChannelEnrollService {
	return &ChannelEnrollService{repo: repo, channels: channels, subs: subs, merchants: merchants, cfg: cfg}
}

// SetSubMerchant 注入微信服务商进件引擎（applyment4sub 提交/媒体上传/查状态）。
// 与代理进件线（EnrollService）共用同一个 SubMerchantService 实例——纯微信对接、无代理逻辑。
func (s *ChannelEnrollService) SetSubMerchant(sm *SubMerchantService) { s.submch = sm }

// ChannelEnrollError 携带业务提示。
type ChannelEnrollError struct{ Msg string }

func (e *ChannelEnrollError) Error() string { return e.Msg }

func ceErr(msg string) *ChannelEnrollError { return &ChannelEnrollError{Msg: msg} }

// channelEnrollStatusText 状态机中文文案。
func channelEnrollStatusText(status string) string {
	switch status {
	case model.ChannelEnrollDraft:
		return "草稿"
	case model.ChannelEnrollPending:
		return "待审核" // 历史存量（旧人工审核态）
	case model.ChannelEnrollSubmitted:
		// 中性词：本地状态机 submitted 覆盖微信侧多个中间态（审核中/待账户验证/待签约/开通权限中）；
		// 用「处理中」避免与微信具体 wx_state（如「待签约」）字面矛盾。
		return "处理中"
	case model.ChannelEnrollApproved:
		return "已开通"
	case model.ChannelEnrollRejected:
		return "已驳回"
	default:
		return status
	}
}

// ChannelEnrollWxStateText 对外暴露：把微信 applyment 状态码转中文，供 handler 在响应里回文案。
// 私有函数 channelEnrollWxStateText 保留供本包内组装视图使用；对外用大写别名，语义等价。
func ChannelEnrollWxStateText(wxState string) string { return channelEnrollWxStateText(wxState) }

// channelEnrollWxStateText 微信 applyment4sub 申请单状态中文文案（对齐官方状态机）。
func channelEnrollWxStateText(wxState string) string {
	switch wxState {
	case "":
		return "—"
	case "APPLYMENT_STATE_EDITTING":
		return "编辑中"
	case "APPLYMENT_STATE_AUDITING":
		return "审核中"
	case "APPLYMENT_STATE_REJECTED":
		return "已驳回"
	case "APPLYMENT_STATE_TO_BE_CONFIRMED":
		return "待账户验证"
	case "APPLYMENT_STATE_TO_BE_SIGNED":
		return "待签约"
	case "APPLYMENT_STATE_SIGNING":
		return "开通权限中"
	case "APPLYMENT_STATE_FINISHED":
		return "已完成"
	case "APPLYMENT_STATE_CANCELED":
		return "已作废"
	default:
		return wxState
	}
}

// channelEnrollMeta 商户线非敏感明文快照落库结构（存 material_meta）。
// = 代理线 EnrollMaterialView（微信五大块非敏感字段 + 敏感 has_*）+ 商户线专属字段。
// material_json 存的是加密后的 applyment4sub 报文（含微信平台公钥密文），不在此结构。
type channelEnrollMeta struct {
	dto.EnrollMaterialView
	ContactPhone string `json:"contact_phone"`
	Remark       string `json:"remark"`
}

// —— 可进件通道（商户端选通道用）——

// EnrollableChannel 可进件的服务商主通道选项。
type EnrollableChannel struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Plugin string `json:"plugin"`
}

// EnrollableChannels 列出所有可进件的服务商通道（启用 + config 含子商户号占位符）。
// 商户端选通道下拉用；直连通道（子商户号为固定值）不进件、不出现。
func (s *ChannelEnrollService) EnrollableChannels() ([]EnrollableChannel, error) {
	list, err := s.channels.ListEnabled()
	if err != nil {
		return nil, err
	}
	out := make([]EnrollableChannel, 0)
	for i := range list {
		if channelHasSubMchPlaceholder(list[i].Config) {
			out = append(out, EnrollableChannel{
				ID: int(list[i].ID), Name: list[i].Name, Plugin: list[i].Plugin,
			})
		}
	}
	return out, nil
}

// —— 列表 / 详情 ——

// List 分页查询进件单，派生主通道名与状态文案。
func (s *ChannelEnrollService) List(q repository.ChannelEnrollQuery) ([]dto.ChannelEnrollView, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	nameCache := map[int]string{}
	phoneCache := map[uint]string{}
	views := make([]dto.ChannelEnrollView, 0, len(list))
	for i := range list {
		views = append(views, s.toView(&list[i], nameCache, phoneCache))
	}
	return views, total, nil
}

// toView 组装列表视图（不含敏感原文）。nameCache/phoneCache 可为 nil（详情单条查询时）。
func (s *ChannelEnrollService) toView(e *model.ChannelEnroll, nameCache map[int]string, phoneCache map[uint]string) dto.ChannelEnrollView {
	name, ok := nameCache[e.ChannelID]
	if !ok {
		if ch, _ := s.channels.FindByID(uint(e.ChannelID)); ch != nil {
			name = ch.Name
		}
		if nameCache != nil {
			nameCache[e.ChannelID] = name
		}
	}
	// 归属商户注册手机：从 pay_merchant 派生（后台列表展示商户手机，与进件时填的 contact_phone 相互独立）。
	merchantPhone, ok := phoneCache[e.UID]
	if !ok {
		if s.merchants != nil {
			if m, _ := s.merchants.FindByUIDSafe(e.UID); m != nil {
				merchantPhone = m.Phone
			}
		}
		if phoneCache != nil {
			phoneCache[e.UID] = merchantPhone
		}
	}
	fmtTime := func(t *time.Time) string {
		if t == nil || t.IsZero() {
			return "—"
		}
		return t.Format(timeLayout)
	}
	// 更新时间：审核完成优先 → 提交时间 → 建单时间（列表「创建/更新时间」双行的第二行）。
	updateTime := e.AddTime.Format(timeLayout)
	if e.SubmitTime != nil && !e.SubmitTime.IsZero() {
		updateTime = e.SubmitTime.Format(timeLayout)
	}
	if e.AuditTime != nil && !e.AuditTime.IsZero() {
		updateTime = e.AuditTime.Format(timeLayout)
	}
	// 支付开关：仅已开通且有子通道的单适用，取子通道启停；其余单 -1（前端显示「—」）。
	subChannelStatus := int8(-1)
	if e.Status == model.ChannelEnrollApproved && e.SubChannelID > 0 {
		if sub, err := s.subs.FindByID(e.SubChannelID); err == nil && sub != nil {
			subChannelStatus = sub.Status
		}
	}
	return dto.ChannelEnrollView{
		ID:           e.ID,
		EnrollNo:     e.EnrollNo,
		UID:          e.UID,
		MerchantName:  e.MerchantName,
		SubjectType:   e.SubjectType,
		ContactPhone:  e.ContactPhone,
		MerchantPhone: merchantPhone,
		ChannelID:     e.ChannelID,
		ChannelName:  name,
		Plugin:       e.Plugin,
		Status:       e.Status,
		StatusText:   channelEnrollStatusText(e.Status),
		SubMchID:      e.SubMchID,
		SubChannelID:  e.SubChannelID,
		RejectReason:  e.RejectReason,
		AuditDetail:   parseChannelEnrollAuditDetail(e.AuditDetail),
		AuditAdmin:    e.AuditAdmin,
		BusinessCode:  e.BusinessCode,
		WxApplymentID: e.WxApplymentID,
		WxState:       e.WxState,
		WxStateText:   channelEnrollWxStateText(e.WxState),
		SignURL:       e.SignURL,
		AddTime:       e.AddTime.Format(timeLayout),
		SubmitTime:    fmtTime(e.SubmitTime),
		AuditTime:     fmtTime(e.AuditTime),
		UpdateTime:       updateTime,
		SubChannelStatus: subChannelStatus,
	}
}

// parseChannelEnrollAuditDetail 把落库的驳回逐字段详情 JSON 串还原为结构化数组（供前端逐项展开）。
// 空串/脏值返回 nil（前端按无逐字段详情处理，只显示整体 reject_reason）。
func parseChannelEnrollAuditDetail(raw string) []dto.ChannelEnrollAuditDetailItem {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var items []dto.ChannelEnrollAuditDetailItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil || len(items) == 0 {
		return nil
	}
	return items
}

// Get 取进件单详情（含填料脱敏回显）。uid>0 时校验归属（商户端只看自己）。
func (s *ChannelEnrollService) Get(id uint, uid uint) (*dto.ChannelEnrollDetail, error) {
	e, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ceErr("进件单不存在")
	}
	if uid > 0 && e.UID != uid {
		return nil, ceErr("该进件单不属于您")
	}
	detail := &dto.ChannelEnrollDetail{
		ChannelEnrollView: s.toView(e, nil, nil),
		Material:          s.materialView(e.MaterialMeta),
	}
	return detail, nil
}

// materialView 从 material_meta（非敏感明文快照）还原脱敏回显（敏感只回 has_*）。
// 空/脏值返回未填视图。material_json 是加密报文不在此解析。
func (s *ChannelEnrollService) materialView(materialMeta string) dto.ChannelEnrollMaterialView {
	v := dto.ChannelEnrollMaterialView{}
	if strings.TrimSpace(materialMeta) == "" {
		return v
	}
	var m channelEnrollMeta
	if err := json.Unmarshal([]byte(materialMeta), &m); err != nil {
		return v
	}
	v.EnrollMaterialView = m.EnrollMaterialView
	v.ContactPhone = m.ContactPhone
	v.Remark = m.Remark
	// 主体名称：营业执照登记名 / 登记证书名 取其一，供列表检索与顶部展示。
	v.MerchantName = firstNonEmpty(m.BusinessMerchantName, m.CertMerchantName)
	// 超管脱敏名兜底：LEGAL 型超管即法人本人，法人姓名是非敏感明文，可据此补出脱敏名
	// （覆盖 ContactNameMasked 字段新增前的历史单）。SUPER（经办人）型无明文可推，保持后端填料时写入的值。
	if v.ContactNameMasked == "" && strings.EqualFold(v.ContactType, "LEGAL") {
		v.ContactNameMasked = maskNameSingle(v.LegalPerson)
	}
	return v
}

// —— 建单（商户端）——

// Create 商户建单：校验主通道存在且为服务商类（config 含 sub_mchid 占位符）→ 复用 draft/rejected 单或新建。
// 同一商户同一通道已有进行中(pending)或已通过(approved)单时拦，避免重复进件。
func (s *ChannelEnrollService) Create(uid uint, req dto.ChannelEnrollCreateReq) (*model.ChannelEnroll, error) {
	ch, err := s.channels.FindByID(uint(req.ChannelID))
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ceErr("归属主通道不存在")
	}
	if !channelHasSubMchPlaceholder(ch.Config) {
		return nil, ceErr("该通道不是服务商进件通道（未配置子商户号占位符），无需进件")
	}
	// 已通过：该通道对该商户已开通，不重复建。
	// 进行中(pending)：已提交待审，不重复建。draft/rejected 可续填复用同一单。
	if reuse, err := s.repo.FindDraftOrRejected(uid, req.ChannelID); err != nil {
		return nil, err
	} else if reuse != nil {
		return reuse, nil
	}
	// 查是否已有 pending/approved（不可重复建）。
	pendings, _, err := s.repo.List(repository.ChannelEnrollQuery{UID: uid, ChannelID: req.ChannelID, Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	for i := range pendings {
		st := pendings[i].Status
		if st == model.ChannelEnrollPending {
			return nil, ceErr("该通道已有待审核的进件单，请勿重复提交")
		}
		if st == model.ChannelEnrollApproved {
			return nil, ceErr("该通道已进件成功并开通，无需重复进件")
		}
	}

	e := &model.ChannelEnroll{
		EnrollNo:     s.genUniqueEnrollNo(),
		UID:          uid,
		ChannelID:    req.ChannelID,
		Plugin:       ch.Plugin,
		MerchantName: strings.TrimSpace(req.MerchantName),
		ContactPhone: strings.TrimSpace(req.ContactPhone),
		Status:       model.ChannelEnrollDraft,
		AddTime:      time.Now(),
	}
	if err := s.repo.Create(e); err != nil {
		return nil, err
	}
	return e, nil
}

// —— 填料（商户端，敏感字段用微信平台公钥 RSA-OAEP 加密）——

// FillMaterial 填/改进件资料。前置：status∈{draft,rejected}（草稿续填 / 驳回重填）。
// ★全自动化（阶段2）：敏感字段用微信平台公钥 RSA-OAEP 加密（EncryptSensitive），组装出完整
//   applyment4sub 报文落 material_json（提交时原样送微信）；非敏感明文快照落 material_meta（回显编辑）。
//   与代理进件线共用 buildApplymentBody 纯函数——同一套字段校验/加密/组装，商户线不走代理端文件。
//   同步把主体名/主体类型/联系手机冗余到单表列供列表检索。uid>0 校验归属。
func (s *ChannelEnrollService) FillMaterial(id uint, uid uint, req dto.ChannelEnrollMaterialReq) (*model.ChannelEnroll, error) {
	e, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ceErr("进件单不存在")
	}
	if uid > 0 && e.UID != uid {
		return nil, ceErr("该进件单不属于您")
	}
	if e.Status != model.ChannelEnrollDraft && e.Status != model.ChannelEnrollRejected {
		return nil, ceErr("当前状态不可填写资料（仅草稿或被驳回后可填）")
	}
	if s.submch == nil || !s.submch.Configured() {
		return nil, ceErr("微信服务商凭证未配置，无法加密敏感资料并提交进件，请联系平台在系统设置填写")
	}
	// 敏感字段用微信平台公钥 RSA-OAEP 加密（进件报文直送微信，无需平台侧解密）。
	enc := func(plain string) (string, error) { return s.submch.EncryptSensitive(strings.TrimSpace(plain)) }

	// 组装 applyment4sub 报文 + 非敏感快照（与代理线共用）。
	body, metaView, err := buildApplymentBody(enc, req.EnrollMaterialReq)
	if err != nil {
		// buildApplymentBody 返回的是 *EnrollError；统一转成本线错误文案。
		return nil, ceErr(err.Error())
	}
	materialJSON, err := json.Marshal(body)
	if err != nil {
		return nil, ceErr("资料组装失败: " + err.Error())
	}
	// 非敏感快照 + 商户线专属字段。
	meta := channelEnrollMeta{
		EnrollMaterialView: metaView,
		ContactPhone:       strings.TrimSpace(req.ContactPhone),
		Remark:             strings.TrimSpace(req.Remark),
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, ceErr("资料快照组装失败: " + err.Error())
	}
	// 主体名称冗余：营业执照登记名 / 登记证书名 取其一。
	name := firstNonEmpty(strings.TrimSpace(req.BusinessMerchantName), strings.TrimSpace(req.CertMerchantName))
	fields := map[string]any{
		"material_json": string(materialJSON),
		"material_meta": string(metaJSON),
		"subject_type":  metaView.SubjectType,
	}
	if name != "" {
		fields["merchant_name"] = name
	}
	if cp := strings.TrimSpace(req.ContactPhone); cp != "" {
		fields["contact_phone"] = cp
	}
	if err := s.repo.Update(id, fields); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

// —— 上传进件资料媒体（商户端，换微信 media_id）——

// UploadMedia 上传一张进件资料图片（营业执照/身份证/门头照等），返回微信 media_id 供表单回填。
// 归属 + 状态校验与 FillMaterial 一致（draft/rejected 可传）；文件类型/大小前置校验后调微信媒体上传。
// 图片二进制不落库，仅换回 media_id。复用代理线同款约束常量（enrollMediaExts/enrollMediaMaxBytes）。
func (s *ChannelEnrollService) UploadMedia(ctx context.Context, id uint, uid uint, filename string, data []byte) (string, error) {
	e, err := s.repo.FindByID(id)
	if err != nil {
		return "", ceErr("进件单不存在")
	}
	if uid > 0 && e.UID != uid {
		return "", ceErr("该进件单不属于您")
	}
	if e.Status != model.ChannelEnrollDraft && e.Status != model.ChannelEnrollRejected {
		return "", ceErr("当前状态不可上传资料（仅草稿或被驳回后可传）")
	}
	if len(data) == 0 {
		return "", ceErr("图片内容为空")
	}
	if len(data) > enrollMediaMaxBytes {
		return "", ceErr("图片不能超过 2M，请压缩后重试")
	}
	if !enrollMediaExts[fileExt(filename)] {
		return "", ceErr("图片仅支持 JPG/PNG/BMP 格式")
	}
	if s.submch == nil || !s.submch.Configured() {
		return "", ceErr("微信服务商凭证未配置，无法上传图片，请联系平台")
	}
	mediaID, err := s.submch.UploadMedia(ctx, filename, data)
	if err != nil {
		var se *SubMchError
		if errors.As(err, &se) {
			return "", ceErr(se.Msg)
		}
		return "", err
	}
	return mediaID, nil
}

// UploadVideo 上传一段进件资料视频（部分行业进件微信要求补充），返回微信 media_id。
// 归属 + 状态校验、文件类型/大小前置校验与图片一致（复用代理线约束常量），通过后调微信视频上传。
func (s *ChannelEnrollService) UploadVideo(ctx context.Context, id uint, uid uint, filename string, data []byte) (string, error) {
	e, err := s.repo.FindByID(id)
	if err != nil {
		return "", ceErr("进件单不存在")
	}
	if uid > 0 && e.UID != uid {
		return "", ceErr("该进件单不属于您")
	}
	if e.Status != model.ChannelEnrollDraft && e.Status != model.ChannelEnrollRejected {
		return "", ceErr("当前状态不可上传资料（仅草稿或被驳回后可传）")
	}
	if len(data) == 0 {
		return "", ceErr("视频内容为空")
	}
	if len(data) > enrollVideoMaxBytes {
		return "", ceErr("视频不能超过 5M，请压缩后重试")
	}
	if !enrollVideoExts[fileExt(filename)] {
		return "", ceErr("视频格式不支持（支持 avi/wmv/mpeg/mp4/mov/mkv/flv/f4v/m4v/rmvb）")
	}
	if s.submch == nil || !s.submch.Configured() {
		return "", ceErr("微信服务商凭证未配置，无法上传视频，请联系平台")
	}
	mediaID, err := s.submch.UploadVideo(ctx, filename, data)
	if err != nil {
		var se *SubMchError
		if errors.As(err, &se) {
			return "", ceErr(se.Msg)
		}
		return "", err
	}
	return mediaID, nil
}

// —— 提交微信（商户端，全自动 applyment4sub）——

// SubmitToWx 把已填资料通过 applyment4sub 直提交微信服务商进件。
// 前置：status∈{draft,rejected} 且已填料。business_code 首次按进件单号生成、驳回重提复用原编号覆盖。
// 提交成功置 submitted + wx_state=AUDITING + 落 wx_applyment_id，之后由 SyncWxState 拉状态推进。uid>0 校验归属。
func (s *ChannelEnrollService) SubmitToWx(ctx context.Context, id uint, uid uint) (*model.ChannelEnroll, error) {
	if s.submch == nil || !s.submch.Configured() {
		return nil, ceErr("微信服务商凭证未配置，请联系平台在系统设置填写")
	}
	e, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ceErr("进件单不存在")
	}
	if uid > 0 && e.UID != uid {
		return nil, ceErr("该进件单不属于您")
	}
	if e.Status != model.ChannelEnrollDraft && e.Status != model.ChannelEnrollRejected {
		return nil, ceErr("当前状态不可提交（仅草稿或被驳回后可提交）")
	}
	if strings.TrimSpace(e.MaterialJSON) == "" {
		return nil, ceErr("请先填写完整进件资料再提交")
	}
	// business_code：驳回重提复用原编号覆盖，首次按进件单号生成（CE 前缀单号已全局唯一）。
	businessCode := e.BusinessCode
	if businessCode == "" {
		businessCode = "CE" + e.EnrollNo
	}
	body, err := injectBusinessCode(e.MaterialJSON, businessCode)
	if err != nil {
		return nil, ceErr("进件资料格式错误: " + err.Error())
	}
	r, _, err := s.submch.SubmitApplyment(ctx, body)
	if err != nil {
		return nil, ceErr(err.Error())
	}
	now := time.Now()
	fields := map[string]any{
		"business_code": businessCode,
		"status":        model.ChannelEnrollSubmitted,
		"wx_state":      "APPLYMENT_STATE_AUDITING",
		"reject_reason": "",
		"audit_detail":  "", // 重新提交清掉上一轮驳回逐字段详情
		"submit_time":   &now,
	}
	if r.ApplymentID > 0 {
		fields["wx_applyment_id"] = decimalToApplyment(r.ApplymentID)
	}
	if err := s.repo.Update(id, fields); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

// SyncWxState 主动拉取微信申请单最新状态并落库，驱动本地状态机推进。
// 终态 FINISHED（拿到 sub_mchid）→ 触发商户线独有的占位符回填交付（非二清落地，钱直清到商户自己的号）；
// REJECTED → 记逐字段驳回详情，商户可改料重提。uid>0 校验归属。★不走代理线的结算/名额逻辑。
func (s *ChannelEnrollService) SyncWxState(ctx context.Context, id uint, uid uint) (*model.ChannelEnroll, error) {
	if s.submch == nil {
		return nil, ceErr("微信进件服务未就绪")
	}
	e, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ceErr("进件单不存在")
	}
	if uid > 0 && e.UID != uid {
		return nil, ceErr("该进件单不属于您")
	}
	if e.Status != model.ChannelEnrollSubmitted {
		return nil, ceErr("仅审核中的进件单可查询微信状态")
	}
	var st *ApplymentStateResp
	if strings.TrimSpace(e.BusinessCode) != "" {
		st, _, err = s.submch.QueryApplymentByBusinessCode(ctx, e.BusinessCode)
	} else if aid := parseApplymentID(e.WxApplymentID); aid > 0 {
		st, _, err = s.submch.QueryApplymentByID(ctx, aid)
	} else {
		return nil, ceErr("该进件单缺少业务申请编号，无法查询")
	}
	if err != nil {
		return nil, ceErr(err.Error())
	}
	return s.applyWxState(e, st)
}

// applyWxState 把微信申请单状态映射到商户线状态机并落库；命中 FINISHED 终态触发占位符回填交付。
// ★与代理线 applyWxState 的关键差异：终态只做「回填子通道占位符」这一步交付（商户线独有），
//   不做代理结算/名额/自动退款——这是「复用微信引擎但不走代理逻辑」的落点。
func (s *ChannelEnrollService) applyWxState(e *model.ChannelEnroll, st *ApplymentStateResp) (*model.ChannelEnroll, error) {
	wxState, wxMsg, subMchID := st.ApplymentState, st.ApplymentStateMsg, st.SubMchID
	fields := map[string]any{"wx_state": wxState}
	if strings.TrimSpace(st.SignURL) != "" {
		fields["sign_url"] = st.SignURL // 超管签约链接（待签约阶段返回，供前端展示扫码）
	}
	if subMchID != "" {
		fields["sub_mchid"] = subMchID // sub_mchid 在 TO_BE_SIGNED/SIGNING/FINISHED 就返回，提前留档
	}

	switch wxState {
	case "APPLYMENT_STATE_FINISHED":
		if subMchID == "" {
			// 状态说完成但未返回 sub_mchid，视为尚未真正开通，保持 submitted 等下次查。
			if err := s.repo.Update(e.ID, fields); err != nil {
				return nil, err
			}
			return s.repo.FindByID(e.ID)
		}
		// ★核心非二清交付：把 sub_mchid 写进该商户该通道的子通道占位符 key，置 status=1。
		//   幂等：仅当此前未 approved 时才建/更新子通道（避免重复查询重复建）。
		if e.Status != model.ChannelEnrollApproved {
			ch, err := s.channels.FindByID(uint(e.ChannelID))
			if err != nil {
				return nil, err
			}
			if ch == nil {
				return nil, ceErr("归属主通道不存在（可能已被删除）")
			}
			subKey := subMchPlaceholderKey(ch.Config)
			if subKey == "" {
				return nil, ceErr("该通道未配置子商户号占位符，无法交付子通道")
			}
			infoKey := placeholderKey(subMchPlaceholderValue(ch.Config, subKey))
			if infoKey == "" {
				infoKey = subKey
			}
			subChannelID, err := s.upsertSubChannel(e, ch.Name, infoKey, subMchID)
			if err != nil {
				return nil, err
			}
			fields["subchannel_id"] = subChannelID
		}
		now := time.Now()
		fields["status"] = model.ChannelEnrollApproved
		fields["audit_time"] = &now
		fields["reject_reason"] = ""
		fields["audit_detail"] = "" // 开通即清掉遗留驳回逐字段详情
		if err := s.repo.Update(e.ID, fields); err != nil {
			return nil, err
		}
	case "APPLYMENT_STATE_REJECTED":
		now := time.Now()
		fields["status"] = model.ChannelEnrollRejected
		fields["reject_reason"] = firstNonEmpty(wxMsg, "微信审核驳回，请按详情修改后重新提交")
		fields["audit_detail"] = marshalAuditDetail(st.AuditDetail)
		fields["audit_time"] = &now
		if err := s.repo.Update(e.ID, fields); err != nil {
			return nil, err
		}
	case "APPLYMENT_STATE_CANCELED":
		// 申请单已作废：转驳回态让商户可重提（商户线无名额概念，不释放冻结）。
		now := time.Now()
		fields["status"] = model.ChannelEnrollRejected
		fields["reject_reason"] = firstNonEmpty(wxMsg, "申请单已作废，请重新提交")
		fields["audit_time"] = &now
		if err := s.repo.Update(e.ID, fields); err != nil {
			return nil, err
		}
	default:
		// 审核中 / 待账户验证 / 待签约 / 开通权限中：保持 submitted，仅刷新 wx_state 与 sign_url。
		if err := s.repo.Update(e.ID, fields); err != nil {
			return nil, err
		}
	}
	return s.repo.FindByID(e.ID)
}

// —— 支付开关（商户端：开关自己已开通渠道的子通道）——

// ToggleSubChannel 商户手动启用/停用自己进件开通的渠道（开关对应子通道 status）。
// 前置：进件单已开通(approved)且已交付子通道(SubChannelID>0)；uid>0 校验进件单与子通道双重归属，
// 商户只能开关自己的。停用后该商户此通道下单命中子通道时被拒（selectchannel 只放行 status=1 子通道）。
func (s *ChannelEnrollService) ToggleSubChannel(id uint, uid uint, enable bool) (*model.ChannelEnroll, error) {
	e, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ceErr("进件单不存在")
	}
	if uid > 0 && e.UID != uid {
		return nil, ceErr("该进件单不属于您")
	}
	if e.Status != model.ChannelEnrollApproved || e.SubChannelID == 0 {
		return nil, ceErr("该渠道尚未开通，无法开关")
	}
	sub, err := s.subs.FindByID(e.SubChannelID)
	if err != nil || sub == nil {
		return nil, ceErr("子通道不存在，请刷新后重试")
	}
	if uid > 0 && sub.UID != uid {
		return nil, ceErr("该渠道不属于您")
	}
	status := int8(0)
	if enable {
		status = 1
	}
	if err := s.subs.Update(sub.ID, map[string]interface{}{"status": status}); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

// —— 手动交付（后台管理员兜底）——

// Approve 管理员手动交付（后台兜底通道）：全自动流程下 SyncWxState 命中 FINISHED 会自动交付，
// 无需此方法；但保留它用于两种兜底场景——① 历史存量 pending 单（旧人工审核态）人工补子商户号；
// ② 微信侧已开通但自动同步异常时管理员手填 sub_mchid 强制交付。
// 交付动作与自动流程一致：把子商户号写进该商户该通道的子通道占位 key、置子通道 status=1、
// 回填本单 SubChannelID(=epay apply_id) 与 sub_mchid，状态→approved（核心非二清交付）。
// 已交付(approved)单拒绝重复。adminName 记操作人。
// Delete 删除进件单（商户端"提交前放弃"入口）。
// 前置：status ∈ {draft, rejected} —— 草稿/被驳回单可删；submitted(审核中)/approved(已开通) 一律拒绝。
// 商户线不走代理侧的 quota 冻结/付费前置，删单无资金或名额副作用，硬删安全。uid>0 校验归属。
func (s *ChannelEnrollService) Delete(id uint, uid uint) error {
	e, err := s.repo.FindByID(id)
	if err != nil {
		return ceErr("进件单不存在")
	}
	if uid > 0 && e.UID != uid {
		return ceErr("该进件单不属于您")
	}
	if e.Status != model.ChannelEnrollDraft && e.Status != model.ChannelEnrollRejected {
		return ceErr("当前状态不可删除（仅草稿或被驳回单可删除）")
	}
	return s.repo.Delete(id)
}

func (s *ChannelEnrollService) Approve(id uint, adminName string, req dto.ChannelEnrollApproveReq) (*model.ChannelEnroll, error) {
	subMchID := strings.TrimSpace(req.SubMchID)
	if subMchID == "" {
		return nil, ceErr("请填写子商户号")
	}
	e, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ceErr("进件单不存在")
	}
	if e.Status == model.ChannelEnrollApproved {
		return nil, ceErr("该进件单已开通交付，无需重复操作")
	}
	if e.Status == model.ChannelEnrollDraft {
		return nil, ceErr("草稿单尚未提交，不能直接交付")
	}
	ch, err := s.channels.FindByID(uint(e.ChannelID))
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ceErr("归属主通道不存在（可能已被删除）")
	}
	// 找到主通道 config 里子商户号的占位 config 键（我方服务商渠道：微信=sub_mchid、富友=appmchid）。
	subKey := subMchPlaceholderKey(ch.Config)
	if subKey == "" {
		return nil, ceErr("该通道未配置子商户号占位符，无法交付子通道")
	}
	// ★写 info 必须用「占位符括号内的 key」而非 config 键名：下单占位替换 mergeSubChannelConfig
	// 取的是 config[k]="[inner]" 的 inner 去 info[inner] 里找值（对齐 epay getSub）。若 config 写成
	// "appmchid":"[fy_mchnt_cd]"，则子商户号要落到 info[fy_mchnt_cd] 才能被替换命中。二者仅在
	// 括号内==键名（如 "sub_mchid":"[sub_mchid]"）时相等，此处按 inner 取值不依赖该巧合。
	infoKey := placeholderKey(subMchPlaceholderValue(ch.Config, subKey))
	if infoKey == "" {
		infoKey = subKey // 兜底：占位值异常时退回键名（不应发生，subKey 已保证是占位符）
	}

	// 建/更新子通道：把 sub_mchid 写进 info[infoKey]，占位符替换生效即钱直清到商户自己的号。
	subChannelID, err := s.upsertSubChannel(e, ch.Name, infoKey, subMchID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.repo.Update(id, map[string]any{
		"status":        model.ChannelEnrollApproved,
		"sub_mchid":     subMchID,
		"subchannel_id": subChannelID,
		"audit_admin":   adminName,
		"audit_time":    &now,
		"reject_reason": "",
		"audit_detail":  "", // 手动交付开通，清掉遗留驳回逐字段详情
	}); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

// upsertSubChannel 审核通过时建或更新子通道：把子商户号写进 info[subKey]，status=1。
// 已由本单开出过子通道（SubChannelID>0 且存在）则更新其 info；否则新建。
// 子通道 info 以现有值为基础合并（保留手工加的其他占位参数），只覆盖 subKey。
func (s *ChannelEnrollService) upsertSubChannel(e *model.ChannelEnroll, channelName, subKey, subMchID string) (uint, error) {
	// 复用已开出的子通道。
	if e.SubChannelID > 0 {
		if sub, err := s.subs.FindByID(e.SubChannelID); err == nil && sub != nil {
			info := mergeInfoKey(sub.Info, subKey, subMchID)
			if err := s.subs.Update(sub.ID, map[string]interface{}{
				"info":     info,
				"status":   int8(1),
				"apply_id": e.ID,
			}); err != nil {
				return 0, err
			}
			return sub.ID, nil
		}
	}
	// 新建子通道：名称取「主通道名-进件单号」保证同商户内唯一。
	info := mergeInfoKey("", subKey, subMchID)
	now := time.Now()
	sub := &model.SubChannel{
		Channel: e.ChannelID,
		UID:     e.UID,
		Name:    channelName + "-" + e.EnrollNo,
		Status:  1, // 进件通过即开通
		Info:    info,
		ApplyID: e.ID,
		AddTime: now,
		UseTime: &now,
	}
	if err := s.subs.Create(sub); err != nil {
		return 0, err
	}
	return sub.ID, nil
}

// Reject 管理员手动驳回（后台兜底）：记原因，状态→rejected，商户可改料重提。
// 全自动流程下驳回由 SyncWxState 命中 REJECTED 自动写入；此方法用于历史 pending 单或人工干预。
// 仅 pending/submitted 可驳回（未交付态）。
func (s *ChannelEnrollService) Reject(id uint, adminName string, req dto.ChannelEnrollRejectReq) (*model.ChannelEnroll, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, ceErr("请填写驳回原因")
	}
	e, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ceErr("进件单不存在")
	}
	if e.Status != model.ChannelEnrollPending && e.Status != model.ChannelEnrollSubmitted {
		return nil, ceErr("仅待审核/审核中的进件单可驳回")
	}
	now := time.Now()
	if err := s.repo.Update(id, map[string]any{
		"status":        model.ChannelEnrollRejected,
		"reject_reason": reason,
		"audit_admin":   adminName,
		"audit_time":    &now,
	}); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

// —— 工具 ——

// mergeInfoKey 把 key=val 合并进现有 info JSON（保留其他键），返回新 JSON 串。
func mergeInfoKey(curInfo, key, val string) string {
	arr := map[string]string{}
	if strings.TrimSpace(curInfo) != "" {
		_ = json.Unmarshal([]byte(curInfo), &arr)
	}
	arr[key] = val
	b, _ := json.Marshal(arr)
	return string(b)
}

// channelHasSubMchPlaceholder 判断主通道 config 是否含子商户号占位符（服务商进件通道判据）。
func channelHasSubMchPlaceholder(config string) bool {
	return subMchPlaceholderKey(config) != ""
}

// subMchPlaceholderKey 从主通道 config 找子商户号占位 key：值形如 "[xxx]" 且 key 命中子商户号候选名
// （sub_mchid/appmchid/submchid，对齐 0730 4.3 各渠道子商户标识字段对照表）。找不到返回空串。
func subMchPlaceholderKey(config string) string {
	if strings.TrimSpace(config) == "" {
		return ""
	}
	cfg := map[string]string{}
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return ""
	}
	candidates := map[string]bool{"sub_mchid": true, "appmchid": true, "submchid": true}
	for k, v := range cfg {
		if candidates[k] && placeholderKey(v) != "" {
			return k
		}
	}
	return ""
}

// subMchPlaceholderValue 取主通道 config 里指定键的原始占位值（如 "[sub_mchid]"）。
// 供 Approve 解出括号内 key 用（写 info 要按括号内 key 落值，见 mergeSubChannelConfig）。
func subMchPlaceholderValue(config, key string) string {
	if strings.TrimSpace(config) == "" || key == "" {
		return ""
	}
	cfg := map[string]string{}
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return ""
	}
	return cfg[key]
}

// genEnrollNoCE 生成进件单号：CE + 10 位随机数字（与代理线 TY 前缀区分）。
func genEnrollNoCE() string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	n := binary.BigEndian.Uint64(append([]byte{0, 0, 0}, b...)) % 10000000000
	return fmt.Sprintf("CE%010d", n)
}

// genUniqueEnrollNo 生成不与库中现有单号冲突的进件单号。
func (s *ChannelEnrollService) genUniqueEnrollNo() string {
	no := genEnrollNoCE()
	for i := 0; i < 8; i++ {
		if _, err := s.repo.FindByNo(no); err != nil {
			return no // 查不到=未占用
		}
		no = genEnrollNoCE()
	}
	return no
}
