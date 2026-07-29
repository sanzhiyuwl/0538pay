package service

import (
	"context"

	"github.com/epvia/api/pkg/ocr"
)

// OCRService 证件识别服务：按系统配置 ocr_provider 选阿里云/腾讯云引擎，
// 供收单侧商户实名认证与代理进件填料共用。识别值仅供前端回填由人核对，不落库。
type OCRService struct {
	cfg *ConfigService
}

// NewOCRService 创建 OCR 服务。
func NewOCRService(cfg *ConfigService) *OCRService {
	return &OCRService{cfg: cfg}
}

// engine 按当前配置构造识别引擎。未开启或缺密钥返回业务错误。
func (s *OCRService) engine() (ocr.Recognizer, error) {
	switch s.cfg.Int("ocr_provider", 0) {
	case 1:
		id, key := s.cfg.Str("ocr_aliyun_id"), s.cfg.Str("ocr_aliyun_key")
		if id == "" || key == "" {
			return nil, &ConfigError{Msg: "阿里云 OCR 未配置 AccessKeyId/AccessKeySecret"}
		}
		return ocr.NewAliyun(id, key), nil
	case 2:
		id, key := s.cfg.Str("ocr_tencent_id"), s.cfg.Str("ocr_tencent_key")
		if id == "" || key == "" {
			return nil, &ConfigError{Msg: "腾讯云 OCR 未配置 SecretId/SecretKey"}
		}
		return ocr.NewTencent(id, key, s.cfg.Str("ocr_tencent_region")), nil
	default:
		return nil, &ConfigError{Msg: "平台未开启 OCR 文字识别，请先在系统设置配置"}
	}
}

// Enabled 当前是否已开启并配好 OCR（前端按此显隐识别按钮）。
func (s *OCRService) Enabled() bool {
	_, err := s.engine()
	return err == nil
}

// RecognizeLicense 识别营业执照。
func (s *OCRService) RecognizeLicense(ctx context.Context, image []byte) (*ocr.LicenseResult, error) {
	eng, err := s.engine()
	if err != nil {
		return nil, err
	}
	r, err := eng.RecognizeBusinessLicense(ctx, image)
	if err != nil {
		return nil, &ConfigError{Msg: "营业执照识别失败：" + err.Error()}
	}
	return r, nil
}

// RecognizeIDCard 识别身份证。side 指定人像面/国徽面。
func (s *OCRService) RecognizeIDCard(ctx context.Context, image []byte, side ocr.IDCardSide) (*ocr.IDCardResult, error) {
	eng, err := s.engine()
	if err != nil {
		return nil, err
	}
	r, err := eng.RecognizeIDCard(ctx, image, side)
	if err != nil {
		return nil, &ConfigError{Msg: "身份证识别失败：" + err.Error()}
	}
	return r, nil
}
