package handler

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// ChannelControlHandler 子商户管控（风控第二段）后台接口。
// 仅后台 /admin：风控总览页读列表/概览、单个/批量刷新微信管控状态、解脱路径主动代办。
type ChannelControlHandler struct {
	svc    *service.ChannelControlService
	submch *service.SubMerchantService // 解脱代办（修改主体资料）表单上传资料图片换 media_id 用
}

func NewChannelControlHandler(svc *service.ChannelControlService, submch *service.SubMerchantService) *ChannelControlHandler {
	return &ChannelControlHandler{svc: svc, submch: submch}
}

func failCC(c *gin.Context, err error) {
	var ce *service.ChannelControlError
	if errors.As(err, &ce) {
		resp.Fail(c, 1121, ce.Msg)
		return
	}
	resp.Fail(c, 1121, "操作失败: "+err.Error())
}

// List GET /api/admin/channel-controls 风控总览（概览 + 已开通子商户管控列表，读本地快照不现查微信）。
func (h *ChannelControlHandler) List(c *gin.Context) {
	res, err := h.svc.List(c.Request.Context())
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}

// Refresh POST /api/admin/channel-controls/:id/refresh 单个商户刷新管控状态（现查微信落快照）。
func (h *ChannelControlHandler) Refresh(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 1121, "进件单号无效")
		return
	}
	res, err := h.svc.RefreshOne(c.Request.Context(), uint(id))
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}

// RefreshAll POST /api/admin/channel-controls/refresh-all 批量刷新全部已开通商户（串行限速）。
func (h *ChannelControlHandler) RefreshAll(c *gin.Context) {
	res, err := h.svc.RefreshAll(c.Request.Context())
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}

// Get GET /api/admin/channel-controls/:id 单个进件单的管控快照（进件详情抽屉「业务受限」就地快照，
// 两处不重复造轮子：与风控总览页共用同一份快照，仅按 enroll_id 取一条）。未开通返回视图为空对象。
func (h *ChannelControlHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 1121, "进件单号无效")
		return
	}
	v, err := h.svc.GetByEnrollID(uint(id))
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, gin.H{"view": v}) // v 可能为 nil（未开通），前端判空
}

// Flows GET /api/admin/channel-controls/:id/flows 某进件单的管控流水时间线（风控第三段处置/订阅回调）。
func (h *ChannelControlHandler) Flows(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 1121, "进件单号无效")
		return
	}
	res, err := h.svc.ListFlows(uint(id))
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}

// —— 解脱路径主动代办（0804 方案补遗：recover_way=修改主体资料/修改结算账户时，服务商直接调 API 代办）——

// genOutRequestNo 生成业务申请编号（时间戳+随机数，非本单幂等键，仅用于本次代办提交；
// 与进件单号生成模式一致，不复用 ChannelEnrollService 私有函数）。
func genOutRequestNo(prefix string) string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	n := binary.BigEndian.Uint64(append([]byte{0, 0, 0}, b...)) % 10000000000
	return fmt.Sprintf("%s%010d", prefix, n)
}

// ModifySettlement POST /api/admin/channel-controls/:id/modify-settlement 代该商户修改结算银行账户。
func (h *ChannelControlHandler) ModifySettlement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 1121, "进件单号无效")
		return
	}
	var req service.ModifySettlementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 1121, "参数错误: "+err.Error())
		return
	}
	appNo, err := h.svc.ModifySettlementFor(c.Request.Context(), uint(id), req)
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, gin.H{"application_no": appNo})
}

// ModifySubjectInfo POST /api/admin/channel-controls/:id/modify-subject 代该商户提交主体资料变更申请。
func (h *ChannelControlHandler) ModifySubjectInfo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 1121, "进件单号无效")
		return
	}
	var req dto.SubjectAlterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 1121, "参数错误: "+err.Error())
		return
	}
	outRequestNo := genOutRequestNo("SA")
	applyID, err := h.svc.ModifySubjectInfoFor(c.Request.Context(), uint(id), outRequestNo, req)
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, gin.H{"apply_id": applyID, "out_request_no": outRequestNo})
}

// UploadMedia POST /api/admin/channel-controls/upload 上传解脱代办（修改主体资料）表单资料图片，换微信 media_id。
// 复用 SubMerchantService 通用媒体上传，无进件单状态限制（进件线的 UploadMedia 只允许 draft/rejected）。
func (h *ChannelControlHandler) UploadMedia(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, 1121, "请选择要上传的图片文件")
		return
	}
	f, err := fh.Open()
	if err != nil {
		resp.Fail(c, 1121, "读取上传文件失败")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		resp.Fail(c, 1121, "读取上传文件失败")
		return
	}
	if len(data) > 2*1024*1024 {
		resp.Fail(c, 1121, "图片不能超过 2M")
		return
	}
	mediaID, err := h.submch.UploadMedia(c.Request.Context(), fh.Filename, data)
	if err != nil {
		resp.Fail(c, 1121, "上传失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"media_id": mediaID})
}
