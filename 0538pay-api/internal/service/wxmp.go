package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/0538pay/api/internal/repository"
)

// WxmpService 微信公众号 API 封装（对齐 epay includes/lib/wechat/WechatAPI.php）。
//
// 覆盖 K-2 甲类可自研骨架：
//   - GetAccessToken：stable_token 换取 + pay_weixin 表缓存（提前 200s 过期，对齐 epay）
//   - SendTemplateMessage：cgi-bin/message/template/send 模板消息
//
// 需公众号 appid/secret 真实凭证才能真发（半乙类）；无凭证时 GetAccessToken 返回明确错误，
// 由 NoticeService 静默降级，不阻塞业务主流程。
type WxmpService struct {
	weixin *repository.WeixinRepo
}

func NewWxmpService(weixin *repository.WeixinRepo) *WxmpService {
	return &WxmpService{weixin: weixin}
}

const wxmpHTTPTimeout = 10 * time.Second

// wxTokenResp stable_token 应答。
type wxTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// GetAccessToken 取公众号 access_token（wid=pay_weixin 主键）。
// 缓存未过期直接返回；否则调 stable_token 刷新并回写缓存（对齐 epay getAccessToken）。
func (s *WxmpService) GetAccessToken(ctx context.Context, wid uint, force bool) (string, error) {
	row, err := s.weixin.FindByID(wid)
	if err != nil {
		return "", err
	}
	if row == nil {
		return "", maErr("公众号记录不存在")
	}
	// 缓存有效（距过期 >200s）且非强制刷新，直接用缓存。
	if !force && row.AccessToken != "" && row.ExpireTime != nil &&
		time.Until(*row.ExpireTime) > 200*time.Second {
		return row.AccessToken, nil
	}
	if row.AppID == "" || row.AppSecret == "" {
		return "", maErr("公众号未配置 appid/appsecret，待真实凭证")
	}
	body, _ := json.Marshal(map[string]string{
		"grant_type": "client_credential",
		"appid":      row.AppID,
		"secret":     row.AppSecret,
	})
	var res wxTokenResp
	if err := wxmpPostJSON(ctx, "https://api.weixin.qq.com/cgi-bin/stable_token", body, &res); err != nil {
		return "", err
	}
	if res.AccessToken == "" {
		return "", maErr("AccessToken 获取失败：" + res.ErrMsg)
	}
	expire := time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)
	now := time.Now()
	_ = s.weixin.Update(wid, map[string]interface{}{
		"access_token": res.AccessToken,
		"update_time":  &now,
		"expire_time":  &expire,
	})
	return res.AccessToken, nil
}

// TplData 模板消息数据项 {key:{value:...}}，对齐微信模板消息 data 结构。
type TplData map[string]struct {
	Value string `json:"value"`
}

// SendTemplateMessage 发送公众号模板消息（对齐 epay sendTemplateMessage）。
func (s *WxmpService) SendTemplateMessage(ctx context.Context, wid uint, openid, templateID, jumpURL string, data TplData) error {
	if openid == "" || templateID == "" {
		return maErr("模板消息缺少 openid 或 template_id")
	}
	token, err := s.GetAccessToken(ctx, wid, false)
	if err != nil {
		return err
	}
	post, _ := json.Marshal(map[string]interface{}{
		"touser":      openid,
		"template_id": templateID,
		"url":         jumpURL,
		"data":        data,
	})
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	url := "https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=" + token
	if err := wxmpPostJSON(ctx, url, post, &res); err != nil {
		return err
	}
	if res.ErrCode != 0 {
		return maErr("模板消息发送失败：" + res.ErrMsg)
	}
	return nil
}

// wxmpPostJSON POST JSON 并解析应答到 out。
func wxmpPostJSON(ctx context.Context, endpoint string, body []byte, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: wxmpHTTPTimeout}).Do(req)
	if err != nil {
		return fmt.Errorf("请求微信接口失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("解析微信应答失败: %w", err)
	}
	return nil
}
