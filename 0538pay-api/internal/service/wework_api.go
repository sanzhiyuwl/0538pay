package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/0538pay/api/internal/model"
)

// 企业微信客服 API 层（K-4，对齐 epay includes/lib/wechat/WeWorkAPI.php）。
//
// 覆盖 access_token 缓存 + 客服账号列表/接待链接 + 消息拉取(syncMsg)/回复(sendMsg) 的 HTTP 骨架。
// 回调 XML 加解密与验签见 pkg/wxwork。真实收发需企业微信 corpid/secret 凭证（半乙类）——
// 无凭证时 GetAccessToken 返回明确错误，上层可优雅提示"待凭证"，不影响 CRUD 配置。
//
// 注：epay WeWorkAPI 还有企业成员消息 send_message / OAuth get_userinfo（网页授权），
// 与客服接待无关，本骨架聚焦客服链路，其余待用到再补。

const weworkHTTPTimeout = 10 * time.Second

// weworkTokenResp qyapi gettoken 应答。
type weworkTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// GetAccessToken 取企微 access_token（wid=pay_wework 主键）。缓存未过期直接返回；
// 否则调 qyapi/gettoken 刷新并回写缓存（对齐 epay WeWorkAPI::getAccessToken，提前 200s 过期）。
func (s *WeworkService) GetAccessToken(ctx context.Context, wid uint, force bool) (string, error) {
	row, err := s.repo.FindByID(wid)
	if err != nil {
		return "", err
	}
	if row == nil {
		return "", wwErr("当前企业微信不存在")
	}
	if !force && row.AccessToken != "" && row.ExpireTime != nil &&
		time.Until(*row.ExpireTime) > 200*time.Second {
		return row.AccessToken, nil
	}
	if row.AppID == "" || row.AppSecret == "" {
		return "", wwErr("企业微信未配置 corpid/secret，待真实凭证")
	}
	u := "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=" + row.AppID + "&corpsecret=" + row.AppSecret
	var res weworkTokenResp
	if err := weworkGetJSON(ctx, u, &res); err != nil {
		return "", err
	}
	if res.AccessToken == "" {
		return "", wwErr("AccessToken 获取失败：" + res.ErrMsg)
	}
	expire := time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)
	now := time.Now()
	_ = s.repo.Update(wid, map[string]interface{}{
		"access_token": res.AccessToken,
		"update_time":  &now,
		"expire_time":  &expire,
	})
	return res.AccessToken, nil
}

// KfAccount 企微客服账号（同步应答项）。
type KfAccount struct {
	OpenKfID string `json:"open_kfid"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
}

// SyncKfList 从企微拉取客服账号列表并 upsert 到本地（对齐 WeWorkAPI::getKFList + 后台「刷新」）。
// 返回本地最新列表。真实拉取需凭证。
func (s *WeworkService) SyncKfList(ctx context.Context, wid uint) ([]model.WxKfAccount, error) {
	token, err := s.GetAccessToken(ctx, wid, false)
	if err != nil {
		return nil, err
	}
	u := "https://qyapi.weixin.qq.com/cgi-bin/kf/account/list?access_token=" + token
	var res struct {
		ErrCode     int         `json:"errcode"`
		ErrMsg      string      `json:"errmsg"`
		AccountList []KfAccount `json:"account_list"`
	}
	if err := weworkPostJSON(ctx, u, map[string]int{"offset": 0, "limit": 100}, &res); err != nil {
		return nil, err
	}
	if res.ErrCode != 0 {
		return nil, wwErr("客服帐号列表获取失败：" + res.ErrMsg)
	}
	for _, a := range res.AccountList {
		_ = s.repo.UpsertKf(&model.WxKfAccount{WID: wid, OpenKfID: a.OpenKfID, Name: a.Name})
	}
	return s.repo.ListKfByWork(wid)
}

// GetKfURL 生成客服接待链接（对齐 WeWorkAPI::getKFURL add_contact_way）。
func (s *WeworkService) GetKfURL(ctx context.Context, wid uint, openKfID, scene string) (string, error) {
	token, err := s.GetAccessToken(ctx, wid, false)
	if err != nil {
		return "", err
	}
	u := "https://qyapi.weixin.qq.com/cgi-bin/kf/add_contact_way?access_token=" + token
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		URL     string `json:"url"`
	}
	if err := weworkPostJSON(ctx, u, map[string]string{"open_kfid": openKfID, "scene": scene}, &res); err != nil {
		return "", err
	}
	if res.ErrCode != 0 || res.URL == "" {
		return "", wwErr("微信客服链接获取失败：" + res.ErrMsg)
	}
	return res.URL, nil
}

// SyncMsg 拉取客服消息（对齐 WeWorkAPI::syncMsg sync_msg）。返回 next_cursor 与原始消息列表。
func (s *WeworkService) SyncMsg(ctx context.Context, wid uint, openKfID, token, cursor string) (string, []json.RawMessage, error) {
	at, err := s.GetAccessToken(ctx, wid, false)
	if err != nil {
		return "", nil, err
	}
	u := "https://qyapi.weixin.qq.com/cgi-bin/kf/sync_msg?access_token=" + at
	var res struct {
		ErrCode    int               `json:"errcode"`
		ErrMsg     string            `json:"errmsg"`
		NextCursor string            `json:"next_cursor"`
		MsgList    []json.RawMessage `json:"msg_list"`
	}
	body := map[string]string{"cursor": cursor, "token": token, "open_kfid": openKfID}
	if err := weworkPostJSON(ctx, u, body, &res); err != nil {
		return "", nil, err
	}
	if res.ErrCode != 0 {
		return "", nil, wwErr("客服消息列表获取失败：" + res.ErrMsg)
	}
	return res.NextCursor, res.MsgList, nil
}

// LockGetMsg 加锁读取最新客服消息：取本地 cursor → syncMsg → 回写 next_cursor
// （对齐 WeWorkAPI::lockGetMsg，cursor 持久化到 pay_wxkfaccount）。
func (s *WeworkService) LockGetMsg(ctx context.Context, wid uint, openKfID, token string) ([]json.RawMessage, error) {
	kf, err := s.repo.FindKfByOpenID(openKfID)
	if err != nil {
		return nil, err
	}
	cursor := ""
	if kf != nil {
		cursor = kf.Cursor
	}
	next, list, err := s.SyncMsg(ctx, wid, openKfID, token, cursor)
	if err != nil {
		return nil, err
	}
	if kf != nil && next != "" {
		_ = s.repo.UpdateKfCursor(kf.ID, next)
	}
	return list, nil
}

// SendMsg 发送客服消息（对齐 WeWorkAPI::sendMsg send_msg）。msgparam 为对应 msgtype 的负载。
func (s *WeworkService) SendMsg(ctx context.Context, wid uint, touser, openKfID, msgtype string, msgparam interface{}) (string, error) {
	token, err := s.GetAccessToken(ctx, wid, false)
	if err != nil {
		return "", err
	}
	u := "https://qyapi.weixin.qq.com/cgi-bin/kf/send_msg?access_token=" + token
	post := map[string]interface{}{"touser": touser, "open_kfid": openKfID, "msgtype": msgtype, msgtype: msgparam}
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		MsgID   string `json:"msgid"`
	}
	if err := weworkPostJSON(ctx, u, post, &res); err != nil {
		return "", err
	}
	if res.ErrCode != 0 {
		return "", wwErr("发送消息失败：" + res.ErrMsg)
	}
	return res.MsgID, nil
}

// SendTextMsg 发送文本客服消息（对齐 WeWorkAPI::sendTextMsg）。
func (s *WeworkService) SendTextMsg(ctx context.Context, wid uint, touser, openKfID, content string) (string, error) {
	return s.SendMsg(ctx, wid, touser, openKfID, "text", map[string]string{"content": content})
}

// ---- HTTP 小工具 ----

func weworkGetJSON(ctx context.Context, u string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return err
	}
	return weworkDo(req, out)
}

func weworkPostJSON(ctx context.Context, u string, body, out interface{}) error {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return weworkDo(req, out)
}

func weworkDo(req *http.Request, out interface{}) error {
	resp, err := (&http.Client{Timeout: weworkHTTPTimeout}).Do(req)
	if err != nil {
		return fmt.Errorf("请求企业微信接口失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("解析企业微信应答失败: %w", err)
	}
	return nil
}
