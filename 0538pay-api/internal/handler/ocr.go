package handler

import (
	"io"

	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/ocr"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// OCRHandler 证件识别（营业执照/身份证）。无状态：只把上传图片交给引擎识别、返回结构化字段，
// 不落库。挂在各鉴权分组下（收单后台/商户中心/代理进件 console+agent），鉴权由分组中间件负责。
type OCRHandler struct {
	svc *service.OCRService
}

// NewOCRHandler 创建 OCR handler。
func NewOCRHandler(svc *service.OCRService) *OCRHandler {
	return &OCRHandler{svc: svc}
}

const ocrMaxUpload = 8 << 20 // 单张识别图上限 8MB

// readOCRImage 从 multipart 表单读 file 字段的图片字节。失败时已写响应。
func readOCRImage(c *gin.Context) ([]byte, bool) {
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, 400, "请选择要识别的图片")
		return nil, false
	}
	if fh.Size > ocrMaxUpload {
		resp.Fail(c, 400, "图片过大，请压缩后重试")
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		resp.Fail(c, 400, "读取上传图片失败")
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		resp.Fail(c, 400, "读取上传图片失败")
		return nil, false
	}
	return data, true
}

// License POST .../ocr/license 识别营业执照，返回结构化字段供前端回填。
func (h *OCRHandler) License(c *gin.Context) {
	data, ok := readOCRImage(c)
	if !ok {
		return
	}
	r, err := h.svc.RecognizeLicense(c.Request.Context(), data)
	if err != nil {
		failOCR(c, err)
		return
	}
	resp.OK(c, r)
}

// IDCard POST .../ocr/idcard 识别身份证。表单可选 side=back 识别国徽面（默认人像面）。
func (h *OCRHandler) IDCard(c *gin.Context) {
	data, ok := readOCRImage(c)
	if !ok {
		return
	}
	side := ocr.IDCardFront
	if c.PostForm("side") == "back" {
		side = ocr.IDCardBack
	}
	r, err := h.svc.RecognizeIDCard(c.Request.Context(), data, side)
	if err != nil {
		failOCR(c, err)
		return
	}
	resp.OK(c, r)
}

// failOCR 统一把 ConfigError（未配置/识别失败）作为业务错误返回，其余作 500。
func failOCR(c *gin.Context, err error) {
	if ce, ok := err.(*service.ConfigError); ok {
		resp.Fail(c, 400, ce.Msg)
		return
	}
	resp.Fail(c, 1500, err.Error())
}
