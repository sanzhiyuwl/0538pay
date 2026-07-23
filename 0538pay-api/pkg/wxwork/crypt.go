// Package wxwork 实现企业微信回调消息的加解密与签名（对齐 epay includes/lib/wechat/WechatCrypt.php + WeWorkMsg.php）。
//
// 加解密：AES-256-CBC，key=base64_decode(EncodingAESKey+"=")(43→32 字节)，iv=key[:16]，
// 明文结构 = 16 位随机串 + 4 字节网络序长度 + 明文 + receiveid(corpid)，PKCS7 补位 block=32。
// 签名：sha1(sort([token, timestamp, nonce, encrypt]))，回调 URL 验签与消息体验签共用。
//
// 纯密码学原语，不引第三方 SDK（与 pkg/wxpayv3 一致）。真实回调收发需企微凭证，此层可独立单测。
package wxwork

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

// Crypt 企微回调加解密器。
type Crypt struct {
	key []byte // AES-256 密钥（32 字节）
	iv  []byte // CBC IV（key 前 16 字节）
}

// NewCrypt 由 EncodingAESKey（43 位 base64）构造。补 "=" 后 base64 解码得 32 字节密钥。
func NewCrypt(encodingAESKey string) (*Crypt, error) {
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, errors.New("EncodingAESKey 非法: " + err.Error())
	}
	if len(key) != 32 {
		return nil, errors.New("EncodingAESKey 解码后应为 32 字节")
	}
	return &Crypt{key: key, iv: key[:16]}, nil
}

// Encrypt 加密明文 data（附带 receiveid=corpid），返回 base64 密文（对齐 WechatCrypt::encrypt）。
func (c *Crypt) Encrypt(data, receiveID string) (string, error) {
	var buf bytes.Buffer
	buf.WriteString(randomStr(16))
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	buf.Write(lenBuf)
	buf.WriteString(data)
	buf.WriteString(receiveID)

	plain := pkcs7Pad(buf.Bytes(), 32)
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	out := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, c.iv).CryptBlocks(out, plain)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt 解密 base64 密文，返回内层明文（XML）。对齐 WechatCrypt::decrypt：
// 去 16 位随机头 → 读 4 字节长度 → 截取该长度明文（receiveid 尾部忽略）。
func (c *Crypt) Decrypt(encrypted string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", errors.New("密文 base64 解码失败")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || len(raw)%aes.BlockSize != 0 {
		return "", errors.New("密文长度非法")
	}
	out := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, c.iv).CryptBlocks(out, raw)
	out = pkcs7Unpad(out)
	if len(out) < 20 {
		return "", errors.New("明文长度不足")
	}
	content := out[16:] // 去 16 位随机串
	msgLen := binary.BigEndian.Uint32(content[:4])
	if int(msgLen) > len(content)-4 {
		return "", errors.New("明文长度字段越界")
	}
	return string(content[4 : 4+msgLen]), nil
}

// Signature 生成回调签名 sha1(sort([token, timestamp, nonce, encrypt]))（对齐 WeWorkMsg::getSignature）。
func Signature(token, timestamp, nonce, encrypt string) string {
	arr := []string{token, timestamp, nonce, encrypt}
	sort.Strings(arr)
	h := sha1.New()
	h.Write([]byte(strings.Join(arr, "")))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 校验回调签名是否匹配（常量含义同 Signature）。
func VerifySignature(token, timestamp, nonce, encrypt, msgSignature string) bool {
	return Signature(token, timestamp, nonce, encrypt) == msgSignature
}

// ---- PKCS7 (block=32，对齐 epay enPKSC7/dePKSC7) ----

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	if pad == 0 {
		pad = blockSize
	}
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > 32 || pad > len(data) {
		return data
	}
	return data[:len(data)-pad]
}

// randomStr 生成 n 位随机 ASCII 串（对齐 WechatCrypt::getRandomStr 的 16 位随机头）。
func randomStr(n int) string {
	const pool = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = pool[int(b[i])%len(pool)]
	}
	return string(b)
}
