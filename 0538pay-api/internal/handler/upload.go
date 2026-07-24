package handler

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// UploadHandler 本地文件上传（对齐 epay admin/ajax.php article_upload）。
// 文件存到 <baseDir>/<subDir>/<md5>.<ext>，对外经静态路由 /uploads/... 访问。
type UploadHandler struct {
	baseDir string // 磁盘根目录，如 "./uploads"
	urlBase string // 对外 URL 前缀，如 "/uploads"
}

// NewUploadHandler 创建上传 handler。baseDir 为磁盘存储根目录。
func NewUploadHandler(baseDir string) *UploadHandler {
	return &UploadHandler{baseDir: baseDir, urlBase: "/uploads"}
}

// 允许的图片扩展名（与 epay 一致）。
var allowedImageExt = map[string]bool{
	"gif": true, "jpg": true, "jpeg": true, "png": true, "bmp": true, "webp": true,
}

const maxUploadSize = 10 << 20 // 单文件上限 10MB

// Image POST /api/admin/upload/image
// 表单字段 file（兼容 imgFile），可选 dir 指定子目录（默认 article）。
// 返回 { url: "/uploads/article/xxx.png" }。
func (h *UploadHandler) Image(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		// 兼容 epay 字段名 imgFile
		fh, err = c.FormFile("imgFile")
	}
	if err != nil {
		resp.Fail(c, 400, "未接收到上传文件")
		return
	}
	if fh.Size > maxUploadSize {
		resp.Fail(c, 400, "文件过大，单张图片不超过 10MB")
		return
	}

	// 校验扩展名（白名单）
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fh.Filename), "."))
	if !allowedImageExt[ext] {
		resp.Fail(c, 400, "不支持的文件类型，仅允许 gif/jpg/jpeg/png/bmp/webp")
		return
	}

	// 子目录白名单，防止路径穿越
	dir := c.PostForm("dir")
	switch dir {
	case "article", "cover", "category", "":
		if dir == "" {
			dir = "article"
		}
	default:
		dir = "article"
	}

	src, err := fh.Open()
	if err != nil {
		resp.Fail(c, 1500, "读取上传文件失败: "+err.Error())
		return
	}
	defer src.Close()

	// 以文件内容 md5 命名，天然去重
	hasher := md5.New()
	if _, err := io.Copy(hasher, src); err != nil {
		resp.Fail(c, 1500, "计算文件指纹失败: "+err.Error())
		return
	}
	name := hex.EncodeToString(hasher.Sum(nil)) + "." + ext

	destDir := filepath.Join(h.baseDir, dir)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		resp.Fail(c, 1500, "创建上传目录失败: "+err.Error())
		return
	}
	destPath := filepath.Join(destDir, name)

	// 已存在同内容文件则跳过写入（md5 去重）
	if _, statErr := os.Stat(destPath); statErr != nil {
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			resp.Fail(c, 1500, "重置文件读取位置失败: "+err.Error())
			return
		}
		dst, err := os.Create(destPath)
		if err != nil {
			resp.Fail(c, 1500, "写入文件失败: "+err.Error())
			return
		}
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			os.Remove(destPath)
			resp.Fail(c, 1500, "写入文件失败: "+err.Error())
			return
		}
		dst.Close()
	}

	url := fmt.Sprintf("%s/%s/%s", h.urlBase, dir, name)
	resp.OK(c, gin.H{"url": url})
}
