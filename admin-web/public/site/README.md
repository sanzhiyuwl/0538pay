# 官网图片素材目录

本目录用于存放官网（/ 首页及子页）的图片素材。位于 `public/` 下，
Vite 会原样映射到网站根路径，**引用时用 `/site/...` 绝对路径**（不经过打包）。

## 目录用途

| 目录 | 放什么 |
|---|---|
| `site/hero/` | 首页 Hero 主视觉、产品截图、mockup 图 |
| `site/features/` | 核心特性 / 场景板块的配图、插画 |
| `site/products/` | 产品矩阵（扫码/公众号/小程序等）相关图 |
| `site/logos/` | 支付渠道 logo、合作伙伴 logo（支付宝/微信/QQ/云闪付等） |
| `site/misc/` | 其它零散素材（背景纹理、装饰图形等） |

## 引用方式

在 Vue 模板里用绝对路径（推荐动态绑定，避免被构建当成模块解析）：

```vue
<!-- 正确：绑定表达式 -->
<img :src="`/site/hero/dashboard.png`" alt="收款概览" />
<img src="/site/logos/alipay.svg" alt="支付宝" onerror="this.style.display='none'" />
```

> 注意：不要用静态字面量 `src="/site/..."` 配合会被 rolldown 当 import 解析的写法；
> 保险起见统一用 `:src` 绑定或普通 `src` + `onerror` 兜底。

## 建议规格

- Hero 截图 / mockup：宽 ≥ 1200px，PNG 或 WebP，可带透明背景。
- 渠道 logo：SVG 优先，或 PNG（尺寸 ≥ 64px，透明底）。
- 统一优先 WebP 压缩，控制单图 < 300KB，保证首页加载速度。
