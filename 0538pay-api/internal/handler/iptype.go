package handler

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// iptype.go —— 真实 IP 探测（对齐 epay includes/functions.php:real_ip + admin/ajax.php:iptype）。
// 后台「其余设置 - IP 获取方式」提供三种取值来源，管理员点探测按钮实时看到三种方式各自解析到的 IP 与归属地，
// 据此选出能显示真实地址的那种，防伪造 IP 请求。

var ipv4Re = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

// isPublicIPv4 校验字符串是公网 IPv4（排除私有/保留段），对齐 epay filter_var 的
// FILTER_FLAG_IPV4|NO_PRIV_RANGE|NO_RES_RANGE。
func isPublicIPv4(s string) bool {
	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil || ip.To4() == nil {
		return false
	}
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	return true
}

// realIP 复刻 epay real_ip($type)：
//
//	type 0：X-Forwarded-For（取首个公网 IPv4）→ Client-IP → CF-Connecting-IP → X-Real-IP → RemoteAddr
//	type 1：CF-Connecting-IP → X-Real-IP → RemoteAddr
//	type 2：仅 RemoteAddr
func realIP(c *gin.Context, typ int) string {
	remote := c.RemoteIP() // 去掉端口后的 RemoteAddr
	ip := remote
	if typ <= 0 {
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			for _, m := range ipv4Re.FindAllString(xff, -1) {
				if isPublicIPv4(m) {
					return m
				}
			}
		}
		if cip := c.GetHeader("Client-IP"); isPublicIPv4(cip) {
			return cip
		}
	}
	if typ <= 1 {
		if cf := c.GetHeader("CF-Connecting-IP"); isPublicIPv4(cf) {
			return cf
		}
		if xr := c.GetHeader("X-Real-IP"); isPublicIPv4(xr) {
			return xr
		}
	}
	return ip
}

// getIPCity 查询 IP 归属地（对齐 epay get_ip_city 走宝塔 get_ip_info）。查询失败或内网返回空串。
func getIPCity(ip string) string {
	if !isPublicIPv4(ip) {
		return ""
	}
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get("https://www.bt.cn/api/panel/get_ip_info?ip=" + ip)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if err != nil {
		return ""
	}
	var out map[string]struct {
		Country  string `json:"country"`
		Province string `json:"province"`
		City     string `json:"city"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return ""
	}
	d, ok := out[ip]
	if !ok {
		return ""
	}
	if d.Country == "中国" {
		return d.Province + d.City
	}
	return d.Country + d.Province + d.City
}

// DetectIPType GET /api/admin/config/iptype 探测三种取值方式各自解析到的真实 IP 与归属地。
// 对齐 epay admin/ajax.php case 'iptype'。
func (h *ConfigHandler) DetectIPType(c *gin.Context) {
	type row struct {
		Name string `json:"name"`
		IP   string `json:"ip"`
		City string `json:"city"`
	}
	names := []string{"0_X_FORWARDED_FOR", "1_X_REAL_IP", "2_REMOTE_ADDR"}
	out := make([]row, 0, 3)
	for typ := 0; typ < 3; typ++ {
		ip := realIP(c, typ)
		out = append(out, row{Name: names[typ], IP: ip, City: getIPCity(ip)})
	}
	resp.OK(c, out)
}
