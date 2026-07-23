package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/model"
)

// parseUint 宽松解析商户号；非法返回 0。
func parseUint(s string) uint {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

// uintToStr 商户号转字符串。
func uintToStr(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}

// genTradeNo 生成系统订单号：YmdHis + 5 位随机（对齐 epay date("YmdHis").rand(11111,99999)）。
func genTradeNo(now time.Time) string {
	return now.Format("20060102150405") + fmt.Sprintf("%05d", rand.Intn(88889)+11111)
}

// hostOf 从 URL 提取主机名，解析失败返回空串（对齐 epay getdomain 语义）。
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// channelConfigKeys 通道 config JSON 里通用字段的键名（各渠道共享；渠道特有键进 Extra）。
// 这些键与前端「配置密钥」表单、通道插件读取保持一致。
var channelConfigKeys = map[string]bool{
	"appid": true, "mch_id": true, "serial_no": true,
	"api_v3_key": true, "private_key": true, "public_key": true,
	"public_key_id": true, "notify_url": true,
}

// buildChannelConfig 把通道表的 config text(JSON) 解析为 channel.Config。
// 通用字段映射到具名字段，其余键保留到 Extra，供渠道插件自取。
// config 为空或非法 JSON 时返回零值 Config（mock 等无需密钥的渠道向后兼容）。
func buildChannelConfig(c *model.Channel) channel.Config {
	if c == nil || c.Config == "" {
		return channel.Config{Extra: map[string]string{}}
	}
	kv := map[string]string{}
	if err := json.Unmarshal([]byte(c.Config), &kv); err != nil {
		return channel.Config{Extra: map[string]string{}}
	}
	return buildChannelConfigFromKV(kv)
}

// mergeSubChannelConfig 用子通道 info 覆盖主通道 config 里形如 "[key]" 的占位键
// （B1-34，对齐 epay Channel::getSub：config 值以 '[' 开头则取 arr[key] 替换）。
// 主 config 或子 info 非法 JSON 时返回 nil（调用方退回主通道 config）。
func mergeSubChannelConfig(mainConfig, subInfo string) map[string]string {
	kv := map[string]string{}
	if err := json.Unmarshal([]byte(mainConfig), &kv); err != nil {
		return nil
	}
	arr := map[string]string{}
	if err := json.Unmarshal([]byte(subInfo), &arr); err != nil {
		return nil
	}
	for k, v := range kv {
		if len(v) >= 2 && v[0] == '[' && v[len(v)-1] == ']' {
			key := v[1 : len(v)-1]
			if rv, ok := arr[key]; ok {
				kv[k] = rv // 子通道自定义值替换占位（各商户各自 appid/mchid 等）
			}
		}
	}
	return kv
}

// buildChannelConfigFromKV 把通道 config 的键值表映射为 channel.Config（子通道占位覆盖后复用此逻辑）。
func buildChannelConfigFromKV(kv map[string]string) channel.Config {
	cfg := channel.Config{Extra: map[string]string{}}
	for k, v := range kv {
		switch k {
		case "appid":
			cfg.AppID = v
		case "mch_id":
			cfg.MchID = v
		case "serial_no":
			cfg.SerialNo = v
		case "api_v3_key":
			cfg.Key = v
		case "private_key":
			cfg.PrivateKey = v
		case "public_key":
			cfg.PublicKey = v
		case "notify_url":
			cfg.NotifyURL = v
		default:
			cfg.Extra[k] = v
		}
	}
	// 未被通用键消费的也可能是别名，统一保留到 Extra（如 public_key_id）
	for k, v := range kv {
		if !channelConfigKeys[k] {
			cfg.Extra[k] = v
		}
	}
	if v, ok := kv["public_key_id"]; ok {
		cfg.Extra["public_key_id"] = v
	}
	return cfg
}

// notifyBackURL 拼接第三方渠道回调地址：base 基址 + "/系统订单号"。
// base 形如 https://pay.example.com/api/pay/notify，最终命中 /api/pay/notify/:trade_no。
// base 为空时返回空串（渠道下单时会因缺 notify_url 报错，提示补配置）。
func notifyBackURL(base, tradeNo string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/" + tradeNo
}
