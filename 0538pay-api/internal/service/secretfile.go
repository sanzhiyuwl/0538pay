package service

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// —— 敏感凭证文件存储（微信服务商私钥/公钥等）——
//
// 安全约定（务必遵守，勿放宽）：
//  1. 密钥文件一律存 secrets/ 目录，该目录★不挂任何 HTTP 静态路由★（区别于公开可下载的 uploads/），
//     且已在 .gitignore 锚定 /secrets/* 不入库——即便拖代码库也拿不到密钥。
//  2. pay_config 里对应键不再存 PEM 明文，而存文件指针，格式 "@file:<相对 secrets 的文件名>"。
//     读取时 resolveSecret 识别该前缀 → 从 secrets/ 读回原文；无前缀则按明文（向后兼容旧数据/迁移期）。
//  3. 文件名固定不可猜（如 wx_partner_private_key.pem），且目录不可经 HTTP 枚举。

// secretFilePrefix 标记 config 值是指向 secrets 目录的文件路径而非明文。
const secretFilePrefix = "@file:"

// secretsDir 敏感凭证文件根目录（相对后端进程工作目录，与 uploads/ 同级但不对外服务）。
var secretsDir = "./secrets"

// isSecretRef 判断 config 值是否为文件指针。
func isSecretRef(v string) bool { return strings.HasPrefix(v, secretFilePrefix) }

// secretRefName 从文件指针取纯文件名（已剥前缀 + basename 防穿越）。
func secretRefName(v string) string {
	return filepath.Base(strings.TrimPrefix(v, secretFilePrefix))
}

// resolveSecret 把 config 原始值解析为真实内容：
// 文件指针 "@file:xxx.pem" → 读 secrets/xxx.pem 内容；否则原样返回（明文/空）。
// 读文件失败返回空串（等价"未配"，上层 Configured() 会拦，不会用错误私钥空跑网关）。
func resolveSecret(raw string) string {
	if !isSecretRef(raw) {
		return raw
	}
	name := secretRefName(raw)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(secretsDir, name))
	if err != nil {
		return ""
	}
	return string(data)
}

// saveSecretFile 把密钥内容写入 secrets/<name>，返回可存入 config 的文件指针 "@file:<name>"。
// 覆盖写（同名直接替换），权限 0600（仅属主可读写）。
func saveSecretFile(name, content string) (string, error) {
	if err := os.MkdirAll(secretsDir, 0o700); err != nil {
		return "", err
	}
	name = filepath.Base(name) // 防路径穿越
	if err := os.WriteFile(filepath.Join(secretsDir, name), []byte(content), 0o600); err != nil {
		return "", err
	}
	return secretFilePrefix + name, nil
}

// secretFingerprint 内容指纹（SHA256 前 12 位十六进制），供脱敏回显时给运营核对"是不是这份"，不泄露原文。
func secretFingerprint(content string) string {
	if content == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])[:12]
}
