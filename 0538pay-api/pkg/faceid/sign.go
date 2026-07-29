package faceid

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// call 发起一次腾讯云 FaceID 调用，返回响应体。TC3-HMAC-SHA256 签名 1:1 对齐 epay QcloudFaceid。
func (c *Client) call(ctx context.Context, action string, param map[string]string) ([]byte, error) {
	if c.secretID == "" || c.secretKey == "" {
		return nil, fmt.Errorf("腾讯云实名核身未配置 SecretId/SecretKey")
	}
	payload, _ := json.Marshal(param)

	now := c.now().UTC()
	timestamp := now.Unix()
	date := now.Format("2006-01-02")

	// 1. 规范请求串
	canonicalHeaders := fmt.Sprintf("content-type:application/json; charset=utf-8\nhost:%s\n", c.host)
	signedHeaders := "content-type;host"
	hashedPayload := sha256Hex(payload)
	canonicalRequest := strings.Join([]string{
		http.MethodPost, "/", "", canonicalHeaders, signedHeaders, hashedPayload,
	}, "\n")

	// 2. 待签名字符串
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, c.service)
	stringToSign := strings.Join([]string{
		"TC3-HMAC-SHA256",
		fmt.Sprintf("%d", timestamp),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// 3. 计算签名
	secretDate := hmacSHA256([]byte("TC3"+c.secretKey), date)
	secretService := hmacSHA256(secretDate, c.service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	// 4. 组装 Authorization
	authorization := fmt.Sprintf(
		"TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.secretID, credentialScope, signedHeaders, signature,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+c.host+"/", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", c.host)
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", c.version)
	req.Header.Set("X-TC-Region", c.region)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("腾讯云实名核身请求失败: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("腾讯云实名核身失败(HTTP %d): %s", res.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
