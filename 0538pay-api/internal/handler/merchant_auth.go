package handler

import (
	"errors"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/middleware"
	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// MerchantAuthHandler 商户端登录 / 当前商户信息。
type MerchantAuthHandler struct {
	svc *service.MerchantAuthService
}

func NewMerchantAuthHandler(svc *service.MerchantAuthService) *MerchantAuthHandler {
	return &MerchantAuthHandler{svc: svc}
}

func failFromMerchantAuthErr(c *gin.Context, err error) {
	var ae *service.MerchantAuthError
	if errors.As(err, &ae) {
		resp.Fail(c, 1101, ae.Msg)
		return
	}
	resp.Fail(c, 1101, "登录失败: "+err.Error())
}

// Login POST /api/merchant/login
func (h *MerchantAuthHandler) Login(c *gin.Context) {
	var req dto.MerchantLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	out, err := h.svc.Login(req)
	if err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, out)
}

// Info GET /api/merchant/info 当前登录商户信息
func (h *MerchantAuthHandler) Info(c *gin.Context) {
	uid, _ := c.Get(middleware.CtxUID)
	id, ok := uid.(uint)
	if !ok || id == 0 {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	info, err := h.svc.CurrentInfo(id)
	if err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, info)
}
