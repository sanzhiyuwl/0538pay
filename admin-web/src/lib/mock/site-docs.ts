/**
 * 开发者文档 CMS 默认数据。
 * 页面由分组、章节和内容块组成，后续可由 store 持久化并交给官网渲染器消费。
 */

export interface DocSettings {
  title: string
  subtitle: string
  defaultSlug: string
}

export interface DocGroup {
  id: string
  name: string
  icon: string
  sort: number
  enabled: boolean
}

export interface DocPage {
  id: number
  groupId: string
  slug: string
  title: string
  keywords: string
  sort: number
  status: 0 | 1
  sections: DocSection[]
}

export interface DocSection {
  id: string
  anchor: string
  title: string
  showInOutline: boolean
  blocks: DocBlock[]
}

interface DocBlockBase {
  id: string
}

export interface DocRichTextBlock extends DocBlockBase {
  type: 'richText'
  html: string
}

export interface DocEndpointBlock extends DocBlockBase {
  type: 'endpoint'
  method: 'GET' | 'POST'
  url: string
}

export interface DocTableColumn {
  key: string
  label: string
  width?: string
  kind?: 'text' | 'code' | 'status'
}

export interface DocTableBlock extends DocBlockBase {
  type: 'table'
  columns: DocTableColumn[]
  rows: Record<string, string>[]
}

export interface DocCodeBlock extends DocBlockBase {
  type: 'code'
  language: string
  code: string
}

export interface DocCalloutBlock extends DocBlockBase {
  type: 'callout'
  tone: 'info' | 'warning' | 'success'
  html: string
}

export interface DocStepsBlock extends DocBlockBase {
  type: 'steps'
  items: string[]
}

export interface DocLinkItem {
  label: string
  url: string
  primary: boolean
}

export interface DocLinksBlock extends DocBlockBase {
  type: 'links'
  items: DocLinkItem[]
}

export interface DocFaqItem {
  q: string
  a: string
}

export interface DocFaqBlock extends DocBlockBase {
  type: 'faq'
  items: DocFaqItem[]
}

export type DocBlock =
  | DocRichTextBlock
  | DocEndpointBlock
  | DocTableBlock
  | DocCodeBlock
  | DocCalloutBlock
  | DocStepsBlock
  | DocLinksBlock
  | DocFaqBlock

export const SITE_DOCS_VERSION = 3

export const defaultDocSettings: DocSettings = {
  title: '开发者文档',
  subtitle: '聚合支付 API 对接指南 · POST + form-urlencoded · 返回 JSON · UTF-8 · SHA256WithRSA',
  defaultSlug: 'intro',
}

export const defaultDocGroups: DocGroup[] = [
  { id: 'system', name: '系统简介', icon: 'FileText', sort: 10, enabled: true },
  { id: 'getting-started', name: '入门', icon: 'Rocket', sort: 20, enabled: true },
  { id: 'payment-v2', name: '支付接口 (V2)', icon: 'CreditCard', sort: 30, enabled: true },
  { id: 'merchant', name: '商户接口', icon: 'Store', sort: 40, enabled: true },
  { id: 'transfer', name: '代付接口', icon: 'ArrowLeftRight', sort: 50, enabled: true },
  { id: 'appendix', name: '附录', icon: 'BookOpen', sort: 60, enabled: true },
  { id: 'help', name: '常见问题', icon: 'HelpCircle', sort: 70, enabled: true },
]

const requestColumns: DocTableColumn[] = [
  { key: 'name', label: '参数', width: '22%', kind: 'code' },
  { key: 'type', label: '类型', width: '12%' },
  { key: 'required', label: '必填', width: '10%', kind: 'status' },
  { key: 'description', label: '说明' },
]

const responseColumns: DocTableColumn[] = [
  { key: 'name', label: '参数', width: '22%', kind: 'code' },
  { key: 'type', label: '类型', width: '12%' },
  { key: 'description', label: '说明' },
]

const commonRequestRows: Record<string, string>[] = [
  { name: 'timestamp', type: 'String', required: '是', description: '当前时间戳，10 位整数，单位秒' },
  { name: 'sign', type: 'String', required: '是', description: '签名字符串，见签名规则' },
  { name: 'sign_type', type: 'String', required: '是', description: '签名类型，默认 RSA' },
]

const submitRequestRows: Record<string, string>[] = [
  { name: 'pid', type: 'Int', required: '是', description: '商户 ID' },
  { name: 'type', type: 'String', required: '否', description: '支付方式；不传则跳收银台自选' },
  { name: 'out_trade_no', type: 'String', required: '是', description: '商户订单号，商户系统内唯一' },
  { name: 'notify_url', type: 'String', required: '是', description: '异步通知地址（服务器）' },
  { name: 'return_url', type: 'String', required: '否', description: '跳转通知地址（页面）' },
  { name: 'name', type: 'String', required: '是', description: '商品名称' },
  { name: 'money', type: 'String', required: '是', description: '金额，单位元，两位小数' },
  { name: 'device', type: 'String', required: '否', description: '设备类型：pc / mobile / qq / wechat / alipay' },
  { name: 'param', type: 'String', required: '否', description: '业务扩展参数，原样返回' },
  ...commonRequestRows,
]

const createRequestRows: Record<string, string>[] = [
  { name: 'pid', type: 'Int', required: '是', description: '商户 ID' },
  { name: 'type', type: 'String', required: '是', description: '支付方式：alipay / wxpay / qqpay / bank' },
  { name: 'out_trade_no', type: 'String', required: '是', description: '商户订单号' },
  { name: 'notify_url', type: 'String', required: '是', description: '异步通知地址' },
  { name: 'return_url', type: 'String', required: '否', description: '跳转通知地址' },
  { name: 'name', type: 'String', required: '是', description: '商品名称' },
  { name: 'money', type: 'String', required: '是', description: '金额，单位元' },
  { name: 'clientip', type: 'String', required: '是', description: '用户发起支付的 IP' },
  { name: 'device', type: 'String', required: '否', description: 'pc / mobile / qq / wechat / alipay' },
  { name: 'param', type: 'String', required: '否', description: '业务扩展参数' },
  ...commonRequestRows,
]

const createResponseRows: Record<string, string>[] = [
  { name: 'code', type: 'Int', description: '0 为成功，其它为失败' },
  { name: 'msg', type: 'String', description: '返回信息' },
  { name: 'trade_no', type: 'String', description: '系统订单号' },
  { name: 'payurl', type: 'String', description: '支付跳转 URL（网页支付时返回）' },
  { name: 'qrcode', type: 'String', description: '二维码链接（扫码支付时返回）' },
  { name: 'urlscheme', type: 'String', description: '小程序跳转（小程序支付时返回）' },
]

const queryRequestRows: Record<string, string>[] = [
  { name: 'pid', type: 'Int', required: '是', description: '商户 ID' },
  { name: 'trade_no', type: 'String', required: '否', description: '系统订单号，与 out_trade_no 二选一' },
  { name: 'out_trade_no', type: 'String', required: '否', description: '商户订单号，与 trade_no 二选一' },
  ...commonRequestRows,
]

const queryResponseRows: Record<string, string>[] = [
  { name: 'code', type: 'Int', description: '0 为成功' },
  { name: 'trade_no', type: 'String', description: '系统订单号' },
  { name: 'out_trade_no', type: 'String', description: '商户订单号' },
  { name: 'api_trade_no', type: 'String', description: '上游接口订单号' },
  { name: 'type', type: 'String', description: '支付方式' },
  { name: 'name', type: 'String', description: '商品名称' },
  { name: 'money', type: 'String', description: '商品金额' },
  { name: 'realmoney', type: 'String', description: '实际支付金额' },
  { name: 'status', type: 'Int', description: '订单状态：0 未支付，1 已支付' },
  { name: 'addtime', type: 'String', description: '创建时间' },
  { name: 'endtime', type: 'String', description: '支付时间' },
  { name: 'param', type: 'String', description: '业务扩展参数' },
]

const refundRequestRows: Record<string, string>[] = [
  { name: 'pid', type: 'Int', required: '是', description: '商户 ID' },
  { name: 'trade_no', type: 'String', required: '否', description: '系统订单号，二选一' },
  { name: 'out_trade_no', type: 'String', required: '否', description: '商户订单号，二选一' },
  { name: 'money', type: 'String', required: '否', description: '退款金额；不传则全额退款' },
  ...commonRequestRows,
]

const notifyRows: Record<string, string>[] = [
  { name: 'pid', type: 'Int', description: '商户 ID' },
  { name: 'trade_no', type: 'String', description: '系统订单号' },
  { name: 'out_trade_no', type: 'String', description: '商户订单号' },
  { name: 'type', type: 'String', description: '支付方式' },
  { name: 'name', type: 'String', description: '商品名称' },
  { name: 'money', type: 'String', description: '商品金额' },
  { name: 'realmoney', type: 'String', description: '实际支付金额（可能返回）' },
  { name: 'trade_status', type: 'String', description: '交易状态：TRADE_SUCCESS' },
  { name: 'param', type: 'String', description: '业务扩展参数' },
  { name: 'sign', type: 'String', description: '签名字符串' },
  { name: 'sign_type', type: 'String', description: '签名类型 RSA' },
]

const merchantInfoResponseRows: Record<string, string>[] = [
  { name: 'code', type: 'Int', description: '0 为成功' },
  { name: 'pid', type: 'Int', description: '商户 ID' },
  { name: 'status', type: 'Int', description: '商户状态：0 已封禁，1 正常，2 待审核' },
  { name: 'pay_status', type: 'Int', description: '支付状态：0 关闭，1 开启' },
  { name: 'settle_status', type: 'Int', description: '结算状态：0 关闭，1 开启' },
  { name: 'money', type: 'String', description: '商户余额，单位元' },
  { name: 'settle_type', type: 'Int', description: '结算方式：1 支付宝 2 微信 3 QQ钱包 4 银行卡' },
  { name: 'settle_account', type: 'String', description: '结算账户' },
  { name: 'settle_name', type: 'String', description: '结算账户姓名' },
  { name: 'order_num', type: 'Int', description: '订单总数量' },
  { name: 'order_num_today', type: 'Int', description: '今日订单数量' },
  { name: 'order_money_today', type: 'String', description: '今日订单收入' },
]

const transferSubmitRequestRows: Record<string, string>[] = [
  { name: 'pid', type: 'Int', required: '是', description: '商户 ID' },
  { name: 'type', type: 'String', required: '是', description: '转账方式：alipay / wxpay / qqpay / bank' },
  { name: 'account', type: 'String', required: '是', description: '收款方账号（支付宝账号/微信 OpenId/银行卡号）' },
  { name: 'name', type: 'String', required: '否', description: '收款方姓名；传入则校验账号与姓名是否匹配' },
  { name: 'money', type: 'String', required: '是', description: '转账金额，单位元' },
  { name: 'remark', type: 'String', required: '否', description: '转账备注' },
  { name: 'out_biz_no', type: 'String', required: '否', description: '转账交易号，19 位纯数字、日期时间开头，防重复' },
  ...commonRequestRows,
]

const transferSubmitResponseRows: Record<string, string>[] = [
  { name: 'code', type: 'Int', description: '0 为成功' },
  { name: 'status', type: 'Int', description: '0 正在处理，1 转账成功' },
  { name: 'out_biz_no', type: 'String', description: '转账交易号，用于后续查询' },
  { name: 'orderid', type: 'String', description: '接口转账单号' },
  { name: 'paydate', type: 'String', description: '转账完成时间' },
  { name: 'cost_money', type: 'String', description: '转账花费金额（从可用余额扣减）' },
]

const balanceResponseRows: Record<string, string>[] = [
  { name: 'code', type: 'Int', description: '0 为成功' },
  { name: 'available_money', type: 'String', description: '商户可用余额，单位元' },
  { name: 'transfer_rate', type: 'String', description: '转账手续费率，单位 %' },
]

const richText = (id: string, html: string): DocRichTextBlock => ({ id, type: 'richText', html })
const endpoint = (id: string, method: 'GET' | 'POST', url: string): DocEndpointBlock => ({ id, type: 'endpoint', method, url })
const table = (id: string, columns: DocTableColumn[], rows: Record<string, string>[]): DocTableBlock => ({ id, type: 'table', columns, rows })
const code = (id: string, language: string, value: string): DocCodeBlock => ({ id, type: 'code', language, code: value })
const callout = (id: string, tone: DocCalloutBlock['tone'], html: string): DocCalloutBlock => ({ id, type: 'callout', tone, html })
const steps = (id: string, items: string[]): DocStepsBlock => ({ id, type: 'steps', items })
const links = (id: string, items: DocLinkItem[]): DocLinksBlock => ({ id, type: 'links', items })
const faq = (id: string, items: DocFaqItem[]): DocFaqBlock => ({ id, type: 'faq', items })

const section = (
  id: string,
  anchor: string,
  title: string,
  blocks: DocBlock[],
  showInOutline = true,
): DocSection => ({ id, anchor, title, showInOutline, blocks })

const page = (
  id: number,
  groupId: string,
  slug: string,
  title: string,
  keywords: string,
  sort: number,
  sections: DocSection[],
): DocPage => ({ id, groupId, slug, title, keywords, sort, status: 1, sections })

const signCode = `// V2 (RSA / SHA256WithRSA) 签名：
// 1. 取所有非空参数，剔除 sign、sign_type
// 2. 按参数名 ASCII 升序排序，拼成 key=value&key=value
// 3. 用【商户私钥】做 SHA256withRSA 签名，得到 sign
$params = [
  'pid'          => 1001,
  'type'         => 'alipay',
  'out_trade_no' => '20260714001',
  'name'         => 'VIP会员',
  'money'        => '9.90',
  'timestamp'    => '1721206072',
];
ksort($params);
$str = urldecode(http_build_query($params));
openssl_sign($str, $sign, $merchantPrivateKey, OPENSSL_ALGO_SHA256);
$sign = base64_encode($sign);

// V1 (MD5)：排序拼接后末尾追加商户密钥
$sign = md5($str . $merchantKey);`

const createCode = `POST {apiurl}api/pay/create
Content-Type: application/x-www-form-urlencoded

pid=1001&type=alipay&out_trade_no=20260714001
&notify_url=https://your.site/notify
&name=VIP会员&money=9.90&clientip=1.2.3.4
&timestamp=1721206072&sign=xxxx&sign_type=RSA

// 返回 JSON
{
  "code": 0, "msg": "success",
  "trade_no": "20260714120000123456",
  "qrcode": "https://qr.alipay.com/xxxx",
  "timestamp": "1721206073", "sign": "xxxx", "sign_type": "RSA"
}`

const notifyCode = `// 平台向 notify_url 以 GET 推送（验签用平台公钥）：
pid=1001&trade_no=20260714120000123456
&out_trade_no=20260714001&type=alipay
&name=VIP会员&money=9.90&trade_status=TRADE_SUCCESS
&timestamp=1721206073&sign=xxxx&sign_type=RSA

// 验签通过后，务必原样输出 success（否则平台按
// 1分钟/3分钟/20分钟/1小时/2小时 重试）
echo 'success';`

const sdkTree = `SDK/
├── index.php            # 接口测试/下单示例页
├── epayapi.php          # 下单接口调用封装
├── notify_url.php       # 异步通知接收 + 验签示例
├── return_url.php       # 同步跳转接收示例
├── query.php            # 订单查询示例
├── refund.php           # 订单退款示例
└── lib/
    ├── epay.config.php      # 配置：apiurl / pid / 平台公钥 / 商户私钥
    └── EpayCore.class.php   # 核心类：签名·验签·请求`

const sdkUsage = `require 'lib/epay.config.php';
require 'lib/EpayCore.class.php';
$epay = new EpayCore($epay_config);

// 1) 页面跳转支付（输出自动提交表单）
$epay->pagePay([
  'type'         => 'alipay',
  'out_trade_no' => date('YmdHis').mt_rand(100, 999),
  'notify_url'   => 'https://your.site/notify_url.php',
  'return_url'   => 'https://your.site/return_url.php',
  'name'         => '支付测试',
  'money'        => '1.00',
]);

// 2) 异步通知验签（notify_url.php）
if ($epay->verify($_GET)) {
  // 验签通过：处理业务（判断金额、幂等），然后：
  echo 'success';
}`

export const defaultDocPages: DocPage[] = [
  page(1, 'system', 'preface', '序言', '序言 文档 阅读指南 接口说明 开发者', 10, [
    section('preface-main', 'preface', '序言', [
      richText('preface-intro', '<p>欢迎使用 0538Pay 开发者文档。本手册面向需要接入聚合收款、订单管理与代付能力的开发者，覆盖从获取密钥、请求签名到支付通知处理的完整流程。</p><p>首次接入建议依次阅读“接入概述”“签名规则”和“支付方式列表”，再根据业务场景选择页面跳转支付或 API 下单。</p>'),
      callout('preface-tip', 'info', '<p>接口地址、商户 ID 与密钥请以商户中心展示为准。生产环境上线前，请完成金额校验、通知验签和订单幂等处理。</p>'),
    ]),
  ]),
  page(2, 'system', 'product-intro', '产品介绍', '产品介绍 聚合支付 收款 订单 退款 代付 商户', 20, [
    section('product-intro-main', 'product-intro', '产品介绍', [
      richText('product-intro-content', '<p>0538Pay 提供统一的聚合支付 API，一次接入即可使用支付宝、微信支付、QQ 钱包及云闪付 / 银行卡等渠道，并统一管理订单、退款、通知与对账。</p><p>平台同时提供页面收银台、服务端 API 下单、商户信息查询和代付接口，适用于网站、移动端、公众号、小程序及企业结算等场景。</p>'),
      steps('product-intro-capabilities', [
        '统一下单：通过页面跳转或 API 创建支付订单。',
        '交易管理：查询订单、发起退款并接收支付结果通知。',
        '商户与代付：查询经营数据、余额并向指定账户转账。',
      ]),
    ]),
  ]),
  page(3, 'getting-started', 'intro', '接入概述', '接入 概述 快速开始 商户 ID 平台公钥 私钥 流程 apiurl', 10, [
    section('intro-main', 'intro', '接入概述', [
      richText('intro-content', '<p>商户注册并实名后，在<strong>“商户中心 → API 信息”</strong>获取商户 ID、平台公钥与商户私钥即可对接。核心流程为：<strong>下单 → 跳转或展示支付 → 异步通知入账 →（可选）查询 / 退款</strong>。</p>'),
      steps('intro-steps', ['获取商户 ID 与 RSA 密钥对', '按签名规则构造请求', '调用下单并处理异步通知']),
      callout('intro-apiurl', 'info', '<p>接口根地址 <code>{apiurl}</code> 以商户后台展示为准。V2 全部为 POST，路径形如 <code>api/pay/create</code>；页面跳转支付除外。</p>'),
    ]),
  ]),
  page(4, 'getting-started', 'sign', '签名规则', '签名 sign rsa md5 sha256 验签 ksort 加密 密钥', 20, [
    section('sign-main', 'sign', '签名规则', [
      richText('sign-content', '<p>取所有非空参数（剔除 <code>sign</code>、<code>sign_type</code>），按参数名 ASCII 升序拼成 <code>key=value&amp;key=value</code>。V2 用商户私钥做 SHA256withRSA 签名、平台公钥验签；V1 末尾追加商户密钥取 MD5。</p>'),
      code('sign-code', 'php', signCode),
    ]),
  ]),
  page(5, 'getting-started', 'paytype', '支付方式列表', '支付方式 type alipay wxpay qqpay bank 支付宝 微信 qq钱包 云闪付 银行卡', 30, [
    section('paytype-main', 'paytype', '支付方式列表', [
      richText('paytype-content', '<p>下单参数 <code>type</code> 可使用以下值。实际可用方式以 <code>api/pay/paytype</code> 返回为准。</p>'),
      table('paytype-table', [
        { key: 'code', label: 'type', width: '40%', kind: 'code' },
        { key: 'name', label: '支付方式' },
      ], [
        { code: 'alipay', name: '支付宝' },
        { code: 'wxpay', name: '微信支付' },
        { code: 'qqpay', name: 'QQ钱包' },
        { code: 'bank', name: '云闪付 / 银行卡' },
      ]),
    ]),
  ]),
  page(6, 'payment-v2', 'pay-submit', '页面跳转支付', '页面跳转 submit 收银台 网页支付 return_url notify_url', 10, [
    section('pay-submit-main', 'pay-submit', '页面跳转支付', [
      endpoint('pay-submit-endpoint', 'GET', '{apiurl}api/pay/submit'),
      richText('pay-submit-content', '<p>用户浏览器跳转到收银台完成支付，适合网页场景。不传 <code>type</code> 则跳聚合收银台自选。</p>'),
      table('pay-submit-params', requestColumns, submitRequestRows),
    ]),
  ]),
  page(7, 'payment-v2', 'pay-create', 'API 下单', 'api 下单 create 服务端 payurl qrcode 二维码 urlscheme 小程序 请求 返回参数', 20, [
    section('pay-create-main', 'pay-create', 'API 下单', [
      endpoint('pay-create-endpoint', 'POST', '{apiurl}api/pay/create'),
      richText('pay-create-content', '<p>服务端下单，返回 <code>payurl</code>、<code>qrcode</code>、<code>urlscheme</code> 之一，由商户自行展示或跳转。</p>'),
    ]),
    section('pay-create-request', 'pay-create-req', '请求参数', [
      table('pay-create-params', requestColumns, createRequestRows),
    ]),
    section('pay-create-response', 'pay-create-resp', '返回参数', [
      table('pay-create-result', responseColumns, createResponseRows),
      code('pay-create-example', 'http', createCode),
    ]),
  ]),
  page(8, 'payment-v2', 'pay-query', '订单查询', '订单查询 query 状态 trade_no out_trade_no 已支付 未支付', 30, [
    section('pay-query-main', 'pay-query', '订单查询', [
      endpoint('pay-query-endpoint', 'POST', '{apiurl}api/pay/query'),
    ]),
    section('pay-query-request', 'pay-query-req', '请求参数', [
      table('pay-query-params', requestColumns, queryRequestRows),
    ]),
    section('pay-query-response', 'pay-query-resp', '返回参数', [
      table('pay-query-result', responseColumns, queryResponseRows),
    ]),
  ]),
  page(9, 'payment-v2', 'pay-refund', '订单退款', '退款 refund 全额 部分退款 退款金额 money', 40, [
    section('pay-refund-main', 'pay-refund', '订单退款', [
      endpoint('pay-refund-endpoint', 'POST', '{apiurl}api/pay/refund'),
      richText('pay-refund-content', '<p>需商户开启退款 API 权限。不传 <code>money</code> 则全额退款，支持部分退款。</p>'),
      table('pay-refund-params', requestColumns, refundRequestRows),
    ]),
  ]),
  page(10, 'payment-v2', 'pay-refundquery', '退款查询', '退款查询 refundquery 退款状态 退款时间', 50, [
    section('pay-refundquery-main', 'pay-refundquery', '退款查询', [
      endpoint('pay-refundquery-endpoint', 'POST', '{apiurl}api/pay/refundquery'),
      richText('pay-refundquery-content', '<p>参数同订单查询（<code>pid</code> + <code>trade_no</code> / <code>out_trade_no</code> 二选一 + 公共参数）。返回退款状态、退款金额、退款时间等。</p>'),
    ]),
  ]),
  page(11, 'payment-v2', 'notify', '异步通知', '异步通知 notify 回调 success trade_status 重试 验签', 60, [
    section('notify-main', 'notify', '异步通知', [
      richText('notify-content', '<p>支付成功后平台向 <code>notify_url</code> 推送结果（GET）。验签通过后须原样返回 <code>success</code>。</p>'),
      table('notify-params', responseColumns, notifyRows),
      code('notify-example', 'php', notifyCode),
      callout('notify-warning', 'warning', '<p>请同时校验订单号、商户 ID 与实际支付金额，并对业务处理做幂等控制。仅在业务处理成功后返回 <code>success</code>。</p>'),
    ]),
  ]),
  page(12, 'payment-v2', 'return', '同步通知', '同步通知 return 跳转 页面 展示', 70, [
    section('return-main', 'return', '同步通知', [
      richText('return-content', '<p>支付完成后浏览器跳转到 <code>return_url</code>，参数同异步通知。<strong>同步通知仅用于页面展示，不可作为到账依据</strong>，请以异步通知为准。</p>'),
    ]),
  ]),
  page(13, 'merchant', 'merchant-info', '商户信息查询', '商户信息 merchant info 余额 结算 状态 pid', 10, [
    section('merchant-info-main', 'merchant-info', '商户信息查询', [
      endpoint('merchant-info-endpoint', 'POST', '{apiurl}api/merchant/info'),
      richText('merchant-info-content', '<p>请求参数为 <code>pid</code> + 公共参数（<code>timestamp</code> / <code>sign</code> / <code>sign_type</code>）。返回参数如下：</p>'),
      table('merchant-info-result', responseColumns, merchantInfoResponseRows),
    ]),
  ]),
  page(14, 'merchant', 'merchant-orders', '订单列表查询', '订单列表 orders 对账 分页 offset limit', 20, [
    section('merchant-orders-main', 'merchant-orders', '订单列表查询', [
      endpoint('merchant-orders-endpoint', 'POST', '{apiurl}api/merchant/orders'),
      richText('merchant-orders-content', '<p>用于对账或同步订单状态。请求参数：<code>pid</code>、<code>offset</code>（从 0 开始）、<code>limit</code>（≤50）、<code>status</code>（可选，0 未支付 / 1 已支付）+ 公共参数。返回 <code>data</code> 订单数组，单条结构同订单查询。</p>'),
    ]),
  ]),
  page(15, 'transfer', 'transfer-submit', '转账发起', '转账 代付 打款 transfer submit account 收款方 out_biz_no', 10, [
    section('transfer-submit-main', 'transfer-submit', '转账发起', [
      endpoint('transfer-submit-endpoint', 'POST', '{apiurl}api/transfer/submit'),
      richText('transfer-submit-content', '<p>需平台开通代付、且商户开启代付 API 开关。返回 <code>status=0</code> 时需稍后调用转账查询。</p>'),
    ]),
    section('transfer-submit-request', 'transfer-submit-req', '请求参数', [
      table('transfer-submit-params', requestColumns, transferSubmitRequestRows),
    ]),
    section('transfer-submit-response', 'transfer-submit-resp', '返回参数', [
      table('transfer-submit-result', responseColumns, transferSubmitResponseRows),
    ]),
  ]),
  page(16, 'transfer', 'transfer-query', '转账查询', '转账查询 代付查询 transfer query 处理中 失败原因', 20, [
    section('transfer-query-main', 'transfer-query', '转账查询', [
      endpoint('transfer-query-endpoint', 'POST', '{apiurl}api/transfer/query'),
      richText('transfer-query-content', '<p>请求参数：<code>pid</code>、<code>out_biz_no</code>（转账交易号）+ 公共参数。返回转账状态（0 处理中 / 1 成功 / 2 失败）、失败原因、接口单号、金额、花费、备注等。</p>'),
    ]),
  ]),
  page(17, 'transfer', 'transfer-balance', '余额查询', '余额查询 balance available_money 手续费率', 30, [
    section('transfer-balance-main', 'transfer-balance', '余额查询', [
      endpoint('transfer-balance-endpoint', 'POST', '{apiurl}api/transfer/balance'),
      table('transfer-balance-result', responseColumns, balanceResponseRows),
    ]),
  ]),
  page(18, 'appendix', 'v1', 'V1 旧版接口 (MD5)', 'v1 旧版 md5 submit.php mapi.php 兼容 老接口', 10, [
    section('v1-main', 'v1', 'V1 旧版接口 (MD5)', [
      richText('v1-content', '<p>旧版使用 MD5 签名，下单地址为 <code>{siteurl}submit.php</code>（页面跳转）与 <code>{siteurl}mapi.php</code>（API 下单）。参数与 V2 基本一致，签名类型固定为 <code>MD5</code>，无 <code>timestamp</code>。异步通知验签通过后返回 <code>success</code>。<strong>新接入建议直接使用 V2。</strong></p>'),
    ]),
  ]),
  page(19, 'appendix', 'errcode', '错误码', '错误码 code -1 -2 签名失败 参数 风控 时间戳', 20, [
    section('errcode-main', 'errcode', '错误码', [
      table('errcode-table', [
        { key: 'code', label: 'code', width: '26%', kind: 'status' },
        { key: 'description', label: '说明' },
      ], [
        { code: '0', description: '成功（V2）' },
        { code: '1', description: '成功（V1）' },
        { code: '-1', description: '签名校验失败' },
        { code: '-2', description: '商户不存在或已禁用' },
        { code: '-3', description: '缺少必要参数' },
        { code: '-4', description: '订单号重复' },
        { code: '-5', description: '金额不合法' },
        { code: '-6', description: '无可用支付通道' },
        { code: '-7', description: '风控拦截' },
        { code: '-8', description: '时间戳过期（±300 秒）' },
      ]),
    ]),
  ]),
  page(20, 'appendix', 'sdk', 'SDK 下载', 'sdk 下载 php demo 示例 epaycore 核心类 config', 30, [
    section('sdk-main', 'sdk', 'SDK 下载', [
      richText('sdk-content', '<p>官方 PHP-SDK（V2.0，RSA 签名）是开箱即用的对接示例包，包含核心类、配置、下单、查询、退款和通知的完整示例。按注释填入商户 ID、平台公钥、商户私钥即可跑通。</p>'),
      links('sdk-links', [
        { label: '下载 PHP-SDK（V2.0 · RSA）', url: '/assets/files/SDK_2.0.zip', primary: true },
        { label: '下载 PHP-SDK（V1 · MD5）', url: '/assets/files/SDK.zip', primary: false },
      ]),
    ]),
    section('sdk-tree-section', 'sdk-tree', 'SDK 目录结构', [
      code('sdk-tree-code', 'text', sdkTree),
    ]),
    section('sdk-usage-section', 'sdk-usage', '核心用法', [
      code('sdk-usage-code', 'php', sdkUsage),
      richText('sdk-usage-content', '<p>核心类 <code>EpayCore</code> 封装了 <code>pagePay()</code> 页面支付、<code>apiPay()</code> API 下单、<code>queryOrder()</code> 查询、<code>refund()</code> 退款、<code>verify()</code> 异步通知验签，签名与验签（SHA256withRSA）已内置，无需自行实现。</p>'),
    ]),
  ]),
  page(21, 'help', 'faq', '常见问题', '常见问题 faq 资质 费率 到账 结算 二清 安全 对接 支付方式', 10, [
    section('faq-main', 'faq', '常见问题', [
      richText('faq-content', '<p>关于接入、费率、资金安全的高频疑问。如仍有疑问，可在商户中心提交工单或联系客服。</p>'),
      faq('faq-list', [
        { q: '接入需要什么资质？', a: '普通商户注册即用，有无营业执照均可；微信 / 支付宝特约商户需要个体户或企业营业执照。具体资质要求见费率方案说明。' },
        { q: '费率是多少？有没有隐藏费用？', a: '费率低至 0.28%，按商户等级与渠道透明计费，无隐藏费用。开通资费与结算费率在费率方案中逐项列明。' },
        { q: '资金安全吗？会不会有二清风险？', a: '本平台是收单外包服务机构，不涉及资金清算、不触碰用户资金。资金由持牌支付机构与你直接清算，不存在二清。' },
        { q: '多久能到账？如何结算？', a: '买家付款秒级入账到你的商户余额，官方 D+1 自动结算至绑定银行卡，也支持手动申请结算。' },
        { q: '技术对接复杂吗？', a: '提供 RESTful API + 完善文档 + 示例代码，支持 MD5 / RSA 双签名，最快 1 天即可完成对接上线。' },
        { q: '支持哪些支付方式？', a: '支持支付宝、微信支付、QQ 钱包、云闪付等主流渠道，覆盖扫码 / 公众号 / 小程序 / H5 / App / 网关 / 企业收付款 / 跨境八大产品。' },
      ]),
    ]),
  ]),
]
