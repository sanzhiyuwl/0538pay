<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BookOpen, ChevronDown, ArrowLeft, ArrowRight } from 'lucide-vue-next'

// 文档目录（分组，锚点滚动）
const groups = [
  {
    group: '入门',
    items: [
      { id: 'intro', title: '接入概述' },
      { id: 'sign', title: '签名规则' },
      { id: 'paytype', title: '支付方式列表' },
    ],
  },
  {
    group: '支付接口 (V2)',
    items: [
      { id: 'pay-submit', title: '页面跳转支付' },
      { id: 'pay-create', title: 'API 下单' },
      { id: 'pay-query', title: '订单查询' },
      { id: 'pay-refund', title: '订单退款' },
      { id: 'pay-refundquery', title: '退款查询' },
      { id: 'notify', title: '异步通知' },
      { id: 'return', title: '同步通知' },
    ],
  },
  {
    group: '商户接口',
    items: [
      { id: 'merchant-info', title: '商户信息查询' },
      { id: 'merchant-orders', title: '订单列表查询' },
    ],
  },
  {
    group: '代付接口',
    items: [
      { id: 'transfer-submit', title: '转账发起' },
      { id: 'transfer-query', title: '转账查询' },
      { id: 'transfer-balance', title: '余额查询' },
    ],
  },
  {
    group: '附录',
    items: [
      { id: 'v1', title: 'V1 旧版接口 (MD5)' },
      { id: 'errcode', title: '错误码' },
      { id: 'sdk', title: 'SDK 下载' },
    ],
  },
]
const flatItems = groups.flatMap((g) => g.items)
// 当前页（分页式：一次只显示一节）。跟随 URL ?p= 便于分享
const route = useRoute()
const router = useRouter()
const validIds = new Set(flatItems.map((i) => i.id))
const active = ref(
  typeof route.query.p === 'string' && validIds.has(route.query.p) ? route.query.p : 'intro',
)

// 分组折叠状态：默认全部折叠，仅展开当前页所在分组
const activeGroup = groups.find((g) => g.items.some((i) => i.id === active.value))?.group
const collapsed = ref<Set<string>>(
  new Set(groups.filter((g) => g.group !== activeGroup).map((g) => g.group)),
)
function toggleGroup(name: string) {
  if (collapsed.value.has(name)) collapsed.value.delete(name)
  else collapsed.value.add(name)
}

// 切页：设当前页 + 同步 URL + 回到顶部
function goPage(id: string) {
  active.value = id
  router.replace({ query: { ...route.query, p: id } })
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 上一页 / 下一页
const activeIndex = computed(() => flatItems.findIndex((i) => i.id === active.value))
const prevItem = computed(() => (activeIndex.value > 0 ? flatItems[activeIndex.value - 1] : null))
const nextItem = computed(() =>
  activeIndex.value < flatItems.length - 1 ? flatItems[activeIndex.value + 1] : null,
)
const activeTitle = computed(() => flatItems[activeIndex.value]?.title ?? '')

// 公共 timestamp/sign/sign_type 行（多个接口复用）
const commonReq = [
  { name: 'timestamp', type: 'String', req: '是', desc: '当前时间戳，10 位整数，单位秒' },
  { name: 'sign', type: 'String', req: '是', desc: '签名字符串，见签名规则' },
  { name: 'sign_type', type: 'String', req: '是', desc: '签名类型，默认 RSA' },
]

// 各接口请求参数
const submitParams = [
  { name: 'pid', type: 'Int', req: '是', desc: '商户 ID' },
  { name: 'type', type: 'String', req: '否', desc: '支付方式；不传则跳收银台自选' },
  { name: 'out_trade_no', type: 'String', req: '是', desc: '商户订单号，商户系统内唯一' },
  { name: 'notify_url', type: 'String', req: '是', desc: '异步通知地址（服务器）' },
  { name: 'return_url', type: 'String', req: '否', desc: '跳转通知地址（页面）' },
  { name: 'name', type: 'String', req: '是', desc: '商品名称' },
  { name: 'money', type: 'String', req: '是', desc: '金额，单位元，两位小数' },
  { name: 'device', type: 'String', req: '否', desc: '设备类型：pc / mobile / qq / wechat / alipay' },
  { name: 'param', type: 'String', req: '否', desc: '业务扩展参数，原样返回' },
  ...commonReq,
]
const createParams = [
  { name: 'pid', type: 'Int', req: '是', desc: '商户 ID' },
  { name: 'type', type: 'String', req: '是', desc: '支付方式：alipay / wxpay / qqpay / bank' },
  { name: 'out_trade_no', type: 'String', req: '是', desc: '商户订单号' },
  { name: 'notify_url', type: 'String', req: '是', desc: '异步通知地址' },
  { name: 'return_url', type: 'String', req: '否', desc: '跳转通知地址' },
  { name: 'name', type: 'String', req: '是', desc: '商品名称' },
  { name: 'money', type: 'String', req: '是', desc: '金额，单位元' },
  { name: 'clientip', type: 'String', req: '是', desc: '用户发起支付的 IP' },
  { name: 'device', type: 'String', req: '否', desc: 'pc / mobile / qq / wechat / alipay' },
  { name: 'param', type: 'String', req: '否', desc: '业务扩展参数' },
  ...commonReq,
]
const createResp = [
  { name: 'code', type: 'Int', desc: '0 为成功，其它为失败' },
  { name: 'msg', type: 'String', desc: '返回信息' },
  { name: 'trade_no', type: 'String', desc: '系统订单号' },
  { name: 'payurl', type: 'String', desc: '支付跳转 URL（网页支付时返回）' },
  { name: 'qrcode', type: 'String', desc: '二维码链接（扫码支付时返回）' },
  { name: 'urlscheme', type: 'String', desc: '小程序跳转（小程序支付时返回）' },
]
const queryParams = [
  { name: 'pid', type: 'Int', req: '是', desc: '商户 ID' },
  { name: 'trade_no', type: 'String', req: '否', desc: '系统订单号，与 out_trade_no 二选一' },
  { name: 'out_trade_no', type: 'String', req: '否', desc: '商户订单号，与 trade_no 二选一' },
  ...commonReq,
]
const queryResp = [
  { name: 'code', type: 'Int', desc: '0 为成功' },
  { name: 'trade_no', type: 'String', desc: '系统订单号' },
  { name: 'out_trade_no', type: 'String', desc: '商户订单号' },
  { name: 'api_trade_no', type: 'String', desc: '上游接口订单号' },
  { name: 'type', type: 'String', desc: '支付方式' },
  { name: 'name', type: 'String', desc: '商品名称' },
  { name: 'money', type: 'String', desc: '商品金额' },
  { name: 'realmoney', type: 'String', desc: '实际支付金额' },
  { name: 'status', type: 'Int', desc: '订单状态：0 未支付，1 已支付' },
  { name: 'addtime', type: 'String', desc: '创建时间' },
  { name: 'endtime', type: 'String', desc: '支付时间' },
  { name: 'param', type: 'String', desc: '业务扩展参数' },
]
const refundParams = [
  { name: 'pid', type: 'Int', req: '是', desc: '商户 ID' },
  { name: 'trade_no', type: 'String', req: '否', desc: '系统订单号，二选一' },
  { name: 'out_trade_no', type: 'String', req: '否', desc: '商户订单号，二选一' },
  { name: 'money', type: 'String', req: '否', desc: '退款金额；不传则全额退款' },
  ...commonReq,
]
const notifyParams = [
  { name: 'pid', type: 'Int', desc: '商户 ID' },
  { name: 'trade_no', type: 'String', desc: '系统订单号' },
  { name: 'out_trade_no', type: 'String', desc: '商户订单号' },
  { name: 'type', type: 'String', desc: '支付方式' },
  { name: 'name', type: 'String', desc: '商品名称' },
  { name: 'money', type: 'String', desc: '商品金额' },
  { name: 'realmoney', type: 'String', desc: '实际支付金额（可能返回）' },
  { name: 'trade_status', type: 'String', desc: '交易状态：TRADE_SUCCESS' },
  { name: 'param', type: 'String', desc: '业务扩展参数' },
  { name: 'sign', type: 'String', desc: '签名字符串' },
  { name: 'sign_type', type: 'String', desc: '签名类型 RSA' },
]
const merchantInfoResp = [
  { name: 'code', type: 'Int', desc: '0 为成功' },
  { name: 'pid', type: 'Int', desc: '商户 ID' },
  { name: 'status', type: 'Int', desc: '商户状态：0 已封禁，1 正常，2 待审核' },
  { name: 'pay_status', type: 'Int', desc: '支付状态：0 关闭，1 开启' },
  { name: 'settle_status', type: 'Int', desc: '结算状态：0 关闭，1 开启' },
  { name: 'money', type: 'String', desc: '商户余额，单位元' },
  { name: 'settle_type', type: 'Int', desc: '结算方式：1 支付宝 2 微信 3 QQ钱包 4 银行卡' },
  { name: 'settle_account', type: 'String', desc: '结算账户' },
  { name: 'settle_name', type: 'String', desc: '结算账户姓名' },
  { name: 'order_num', type: 'Int', desc: '订单总数量' },
  { name: 'order_num_today', type: 'Int', desc: '今日订单数量' },
  { name: 'order_money_today', type: 'String', desc: '今日订单收入' },
]
const transferSubmitParams = [
  { name: 'pid', type: 'Int', req: '是', desc: '商户 ID' },
  { name: 'type', type: 'String', req: '是', desc: '转账方式：alipay / wxpay / qqpay / bank' },
  { name: 'account', type: 'String', req: '是', desc: '收款方账号（支付宝账号/微信OpenId/银行卡号）' },
  { name: 'name', type: 'String', req: '否', desc: '收款方姓名；传入则校验账号与姓名是否匹配' },
  { name: 'money', type: 'String', req: '是', desc: '转账金额，单位元' },
  { name: 'remark', type: 'String', req: '否', desc: '转账备注' },
  { name: 'out_biz_no', type: 'String', req: '否', desc: '转账交易号，19 位纯数字、日期时间开头，防重复' },
  ...commonReq,
]
const transferSubmitResp = [
  { name: 'code', type: 'Int', desc: '0 为成功' },
  { name: 'status', type: 'Int', desc: '0 正在处理，1 转账成功' },
  { name: 'out_biz_no', type: 'String', desc: '转账交易号，用于后续查询' },
  { name: 'orderid', type: 'String', desc: '接口转账单号' },
  { name: 'paydate', type: 'String', desc: '转账完成时间' },
  { name: 'cost_money', type: 'String', desc: '转账花费金额（从可用余额扣减）' },
]
const balanceResp = [
  { name: 'code', type: 'Int', desc: '0 为成功' },
  { name: 'available_money', type: 'String', desc: '商户可用余额，单位元' },
  { name: 'transfer_rate', type: 'String', desc: '转账手续费率，单位 %' },
]

// 支付方式
const payTypeList = [
  { code: 'alipay', name: '支付宝' },
  { code: 'wxpay', name: '微信支付' },
  { code: 'qqpay', name: 'QQ钱包' },
  { code: 'bank', name: '云闪付 / 银行卡' },
]

// 错误码
const errcodes = [
  { code: '0', text: '成功（V2）', ok: true },
  { code: '1', text: '成功（V1）', ok: true },
  { code: '-1', text: '签名校验失败', ok: false },
  { code: '-2', text: '商户不存在或已禁用', ok: false },
  { code: '-3', text: '缺少必要参数', ok: false },
  { code: '-4', text: '订单号重复', ok: false },
  { code: '-5', text: '金额不合法', ok: false },
  { code: '-6', text: '无可用支付通道', ok: false },
  { code: '-7', text: '风控拦截', ok: false },
  { code: '-8', text: '时间戳过期（±300 秒）', ok: false },
]

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

const notifyCode = `// 平台向 notify_url 以 GET 推送（验签用平台公钥）：
pid=1001&trade_no=20260714120000123456
&out_trade_no=20260714001&type=alipay
&name=VIP会员&money=9.90&trade_status=TRADE_SUCCESS
&timestamp=1721206073&sign=xxxx&sign_type=RSA

// 验签通过后，务必原样输出 success（否则平台按
// 1分钟/3分钟/20分钟/1小时/2小时 重试）
echo 'success';`
</script>

<template>
  <div class="site-surface flex min-h-[calc(100vh-4rem)]">
    <!-- 左侧目录（贴最左，固定栏） -->
    <aside class="hidden w-56 shrink-0 border-r border-border bg-background lg:block">
      <div class="docs-nav sticky top-16 overflow-y-auto py-6 [scrollbar-gutter:stable] [max-height:calc(100vh-4rem)]">
        <div class="flex items-center gap-2 px-6 pb-4">
          <div class="flex size-8 items-center justify-center rounded-lg bg-primary/[0.08] text-primary">
            <BookOpen class="size-[18px]" />
          </div>
          <div class="text-sm font-semibold">开发者文档</div>
        </div>
        <nav class="space-y-1">
          <div v-for="g in groups" :key="g.group">
            <!-- 分组标题（可折叠） -->
            <button
              class="flex w-full items-center px-6 py-1.5 text-xs font-semibold uppercase tracking-wide text-muted-foreground/70 transition-colors hover:text-foreground"
              @click="toggleGroup(g.group)"
            >
              <span class="flex-1 text-left">{{ g.group }}</span>
              <ChevronDown class="size-3.5 transition-transform" :class="collapsed.has(g.group) && '-rotate-90'" />
            </button>
            <!-- 分组条目 -->
            <div v-show="!collapsed.has(g.group)" class="pb-1.5">
              <button
                v-for="t in g.items"
                :key="t.id"
                class="block w-full border-l-2 px-6 py-1.5 text-left text-[13px] transition-colors"
                :class="active === t.id
                  ? 'border-primary bg-primary/[0.08] font-medium text-primary'
                  : 'border-transparent text-muted-foreground hover:bg-accent hover:text-foreground'"
                @click="goPage(t.id)"
              >
                {{ t.title }}
              </button>
            </div>
          </div>
        </nav>
      </div>
    </aside>

    <!-- 右侧内容 -->
    <div class="min-w-0 flex-1">
      <div class="space-y-16 px-8 py-12 lg:px-14 xl:pr-24">
        <header>
          <div class="flex items-center gap-1.5 text-sm text-muted-foreground">
            <span>开发者文档</span>
            <ArrowRight class="size-3.5" />
            <span class="text-foreground">{{ activeTitle }}</span>
          </div>
          <p class="mt-2 text-xs text-muted-foreground">聚合支付 API 对接指南 · POST + form-urlencoded · 返回 JSON · UTF-8 · SHA256WithRSA</p>
        </header>

        <!-- 接入概述 -->
        <section v-show="active === 'intro'" id="intro" class="scroll-mt-20">
          <h2 class="doc-h2">接入概述</h2>
          <p class="doc-p">
            商户注册并实名后，在<b class="text-foreground">「商户中心 → API 信息」</b>获取商户 ID、平台公钥与商户私钥即可对接。
            核心流程：<b class="text-foreground">下单 → 跳转/展示支付 → 异步通知入账 → （可选）查询 / 退款</b>。
          </p>
          <div class="mt-4 grid gap-3 sm:grid-cols-3">
            <div v-for="s in ['1. 获取商户 ID 与 RSA 密钥对', '2. 按签名规则构造请求', '3. 调用下单并处理异步通知']" :key="s" class="rounded-lg border border-border bg-background p-4 text-sm">{{ s }}</div>
          </div>
          <div class="mt-4 rounded-lg border border-border bg-background p-4 text-sm text-muted-foreground">
            接口根地址 <code class="doc-inline">{apiurl}</code> 以商户后台展示为准。V2 全部为 POST，路径形如 <code class="doc-inline">api/pay/create</code>。
          </div>
        </section>

        <!-- 签名规则 -->
        <section v-show="active === 'sign'" id="sign" class="scroll-mt-20">
          <h2 class="doc-h2">签名规则</h2>
          <p class="doc-p">
            取所有非空参数（剔除 <code class="doc-inline">sign</code>、<code class="doc-inline">sign_type</code>），按参数名 ASCII 升序拼成
            <code class="doc-inline">key=value&key=value</code>。V2 用商户私钥做 SHA256withRSA 签名、平台公钥验签；V1 末尾追加商户密钥取 MD5。
          </p>
          <pre class="doc-code"><code>{{ signCode }}</code></pre>
        </section>

        <!-- 支付方式列表 -->
        <section v-show="active === 'paytype'" id="paytype" class="scroll-mt-20">
          <h2 class="doc-h2">支付方式列表</h2>
          <p class="doc-p">下单参数 <code class="doc-inline">type</code> 取值。实际可用方式以 <code class="doc-inline">api/pay/paytype</code> 返回为准。</p>
          <div class="doc-table-wrap">
            <table class="tbl w-full max-w-md">
              <thead><tr><th class="w-[40%]">type</th><th>支付方式</th></tr></thead>
              <tbody>
                <tr v-for="p in payTypeList" :key="p.code"><td class="font-mono text-primary">{{ p.code }}</td><td class="text-muted-foreground">{{ p.name }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 页面跳转支付 -->
        <section v-show="active === 'pay-submit'" id="pay-submit" class="scroll-mt-20">
          <h2 class="doc-h2">页面跳转支付</h2>
          <div class="doc-meta"><span class="doc-method">GET</span><code class="doc-url">{apiurl}api/pay/submit</code></div>
          <p class="doc-p">用户浏览器跳转到收银台完成支付，适合网页场景。不传 <code class="doc-inline">type</code> 则跳聚合收银台自选。</p>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[22%]">参数</th><th class="w-[12%]">类型</th><th class="w-[10%]">必填</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in submitParams" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td>{{ p.req }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- API下单 -->
        <section v-show="active === 'pay-create'" id="pay-create" class="scroll-mt-20">
          <h2 class="doc-h2">API 下单</h2>
          <div class="doc-meta"><span class="doc-method">POST</span><code class="doc-url">{apiurl}api/pay/create</code></div>
          <p class="doc-p">服务端下单，返回 payurl / qrcode / urlscheme 之一，由商户自行展示或跳转。</p>
          <h3 class="doc-h3">请求参数</h3>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[22%]">参数</th><th class="w-[12%]">类型</th><th class="w-[10%]">必填</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in createParams" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td>{{ p.req }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
          <h3 class="doc-h3">返回参数</h3>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[22%]">参数</th><th class="w-[12%]">类型</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in createResp" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
          <pre class="doc-code mt-4"><code>{{ createCode }}</code></pre>
        </section>

        <!-- 订单查询 -->
        <section v-show="active === 'pay-query'" id="pay-query" class="scroll-mt-20">
          <h2 class="doc-h2">订单查询</h2>
          <div class="doc-meta"><span class="doc-method">POST</span><code class="doc-url">{apiurl}api/pay/query</code></div>
          <h3 class="doc-h3">请求参数</h3>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[22%]">参数</th><th class="w-[12%]">类型</th><th class="w-[10%]">必填</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in queryParams" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td>{{ p.req }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
          <h3 class="doc-h3">返回参数</h3>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[22%]">参数</th><th class="w-[12%]">类型</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in queryResp" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 订单退款 -->
        <section v-show="active === 'pay-refund'" id="pay-refund" class="scroll-mt-20">
          <h2 class="doc-h2">订单退款</h2>
          <div class="doc-meta"><span class="doc-method">POST</span><code class="doc-url">{apiurl}api/pay/refund</code></div>
          <p class="doc-p">需商户开启退款 API 权限。不传 <code class="doc-inline">money</code> 则全额退款，支持部分退款。</p>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[22%]">参数</th><th class="w-[12%]">类型</th><th class="w-[10%]">必填</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in refundParams" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td>{{ p.req }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 退款查询 -->
        <section v-show="active === 'pay-refundquery'" id="pay-refundquery" class="scroll-mt-20">
          <h2 class="doc-h2">退款查询</h2>
          <div class="doc-meta"><span class="doc-method">POST</span><code class="doc-url">{apiurl}api/pay/refundquery</code></div>
          <p class="doc-p">参数同订单查询（pid + trade_no/out_trade_no 二选一 + 公共参数）。返回退款状态、退款金额、退款时间等。</p>
        </section>

        <!-- 异步通知 -->
        <section v-show="active === 'notify'" id="notify" class="scroll-mt-20">
          <h2 class="doc-h2">异步通知</h2>
          <p class="doc-p">支付成功后平台向 <code class="doc-inline">notify_url</code> 推送结果（GET）。验签通过后须原样返回 <code class="doc-inline">success</code>。</p>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[22%]">参数</th><th class="w-[12%]">类型</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in notifyParams" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
          <pre class="doc-code mt-4"><code>{{ notifyCode }}</code></pre>
        </section>

        <!-- 同步通知 -->
        <section v-show="active === 'return'" id="return" class="scroll-mt-20">
          <h2 class="doc-h2">同步通知</h2>
          <p class="doc-p">
            支付完成后浏览器跳转到 <code class="doc-inline">return_url</code>，参数同异步通知。
            <b class="text-foreground">仅用于页面展示，不可作为到账依据</b>，请以异步通知为准。
          </p>
        </section>

        <!-- 商户信息 -->
        <section v-show="active === 'merchant-info'" id="merchant-info" class="scroll-mt-20">
          <h2 class="doc-h2">商户信息查询</h2>
          <div class="doc-meta"><span class="doc-method">POST</span><code class="doc-url">{apiurl}api/merchant/info</code></div>
          <p class="doc-p">请求参数：pid + 公共参数（timestamp / sign / sign_type）。返回：</p>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[24%]">参数</th><th class="w-[12%]">类型</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in merchantInfoResp" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 订单列表 -->
        <section v-show="active === 'merchant-orders'" id="merchant-orders" class="scroll-mt-20">
          <h2 class="doc-h2">订单列表查询</h2>
          <div class="doc-meta"><span class="doc-method">POST</span><code class="doc-url">{apiurl}api/merchant/orders</code></div>
          <p class="doc-p">
            用于对账或同步订单状态。请求参数：pid、<code class="doc-inline">offset</code>（从 0 开始）、<code class="doc-inline">limit</code>（≤50）、
            <code class="doc-inline">status</code>（可选，0 未支付 / 1 已支付）+ 公共参数。返回 <code class="doc-inline">data</code> 订单数组，单条结构同订单查询。
          </p>
        </section>

        <!-- 转账发起 -->
        <section v-show="active === 'transfer-submit'" id="transfer-submit" class="scroll-mt-20">
          <h2 class="doc-h2">转账发起</h2>
          <div class="doc-meta"><span class="doc-method">POST</span><code class="doc-url">{apiurl}api/transfer/submit</code></div>
          <p class="doc-p">需平台开通代付、且商户开启代付 API 开关。返回 status=0 时需稍后调用转账查询。</p>
          <h3 class="doc-h3">请求参数</h3>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[22%]">参数</th><th class="w-[12%]">类型</th><th class="w-[10%]">必填</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in transferSubmitParams" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td>{{ p.req }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
          <h3 class="doc-h3">返回参数</h3>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[22%]">参数</th><th class="w-[12%]">类型</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in transferSubmitResp" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 转账查询 -->
        <section v-show="active === 'transfer-query'" id="transfer-query" class="scroll-mt-20">
          <h2 class="doc-h2">转账查询</h2>
          <div class="doc-meta"><span class="doc-method">POST</span><code class="doc-url">{apiurl}api/transfer/query</code></div>
          <p class="doc-p">
            请求参数：pid、<code class="doc-inline">out_biz_no</code>（转账交易号）+ 公共参数。
            返回转账状态（0 处理中 / 1 成功 / 2 失败）、失败原因、接口单号、金额、花费、备注等。
          </p>
        </section>

        <!-- 余额查询 -->
        <section v-show="active === 'transfer-balance'" id="transfer-balance" class="scroll-mt-20">
          <h2 class="doc-h2">余额查询</h2>
          <div class="doc-meta"><span class="doc-method">POST</span><code class="doc-url">{apiurl}api/transfer/balance</code></div>
          <div class="doc-table-wrap">
            <table class="tbl w-full">
              <thead><tr><th class="w-[24%]">参数</th><th class="w-[12%]">类型</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="p in balanceResp" :key="p.name"><td class="font-mono text-[13px] text-primary">{{ p.name }}</td><td class="text-muted-foreground">{{ p.type }}</td><td class="text-muted-foreground">{{ p.desc }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- V1 旧版 -->
        <section v-show="active === 'v1'" id="v1" class="scroll-mt-20">
          <h2 class="doc-h2">V1 旧版接口 (MD5)</h2>
          <p class="doc-p">
            旧版使用 MD5 签名，下单地址为 <code class="doc-inline">{siteurl}submit.php</code>（页面跳转）与
            <code class="doc-inline">{siteurl}mapi.php</code>（API 下单）。参数与 V2 基本一致，签名类型固定
            <code class="doc-inline">MD5</code>，无 timestamp。异步通知验签通过后返回 <code class="doc-inline">success</code>。
            <b class="text-foreground">新接入建议直接使用 V2。</b>
          </p>
        </section>

        <!-- 错误码 -->
        <section v-show="active === 'errcode'" id="errcode" class="scroll-mt-20">
          <h2 class="doc-h2">错误码</h2>
          <div class="doc-table-wrap">
            <table class="tbl w-full max-w-md">
              <thead><tr><th class="w-[26%]">code</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="e in errcodes" :key="e.code"><td class="font-mono" :class="e.ok ? 'text-success' : 'text-destructive'">{{ e.code }}</td><td class="text-muted-foreground">{{ e.text }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- SDK -->
        <section v-show="active === 'sdk'" id="sdk" class="scroll-mt-20">
          <h2 class="doc-h2">SDK 下载</h2>
          <p class="doc-p">
            官方 PHP-SDK（V2.0，RSA 签名），开箱即用的对接示例包。含核心类、配置、下单/查询/退款/通知的完整示例，
            按注释填入商户 ID、平台公钥、商户私钥即可跑通。
          </p>
          <div class="mt-4 flex flex-wrap gap-3">
            <a href="/assets/files/SDK_2.0.zip" class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90">
              <BookOpen class="size-4" />下载 PHP-SDK（V2.0 · RSA）
            </a>
            <a href="/assets/files/SDK.zip" class="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-4 py-2 text-sm transition-colors hover:border-primary/40 hover:text-primary">
              <BookOpen class="size-4" />下载 PHP-SDK（V1 · MD5）
            </a>
          </div>

          <h3 class="doc-h3">SDK 目录结构</h3>
          <pre class="doc-code"><code>{{ sdkTree }}</code></pre>

          <h3 class="doc-h3">核心用法</h3>
          <pre class="doc-code"><code>{{ sdkUsage }}</code></pre>
          <p class="doc-p">
            核心类 <code class="doc-inline">EpayCore</code> 封装了 <code class="doc-inline">pagePay()</code> 页面支付、
            <code class="doc-inline">apiPay()</code> API 下单、<code class="doc-inline">queryOrder()</code> 查询、
            <code class="doc-inline">refund()</code> 退款、<code class="doc-inline">verify()</code> 异步通知验签，
            签名与验签（SHA256withRSA）已内置，无需自行实现。
          </p>
        </section>

        <!-- 上一页 / 下一页 -->
        <nav class="flex items-center justify-between border-t border-border pt-6">
          <button
            v-if="prevItem"
            class="group flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-primary"
            @click="goPage(prevItem.id)"
          >
            <ArrowLeft class="size-4 transition-transform group-hover:-translate-x-0.5" />
            <span>{{ prevItem.title }}</span>
          </button>
          <span v-else />
          <button
            v-if="nextItem"
            class="group flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-primary"
            @click="goPage(nextItem.id)"
          >
            <span>{{ nextItem.title }}</span>
            <ArrowRight class="size-4 transition-transform group-hover:translate-x-0.5" />
          </button>
        </nav>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 侧栏导航：内容超出一屏时，顶部/底部柔和渐隐，不生硬截断 */
.docs-nav {
  -webkit-mask-image: linear-gradient(to bottom, transparent 0, #000 16px, #000 calc(100% - 20px), transparent 100%);
  mask-image: linear-gradient(to bottom, transparent 0, #000 16px, #000 calc(100% - 20px), transparent 100%);
}
.doc-h2 {
  font-size: 1.35rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  scroll-margin-top: 5rem;
}
.doc-h3 {
  margin-top: 1.5rem;
  margin-bottom: 0.25rem;
  font-size: 0.95rem;
  font-weight: 600;
}
.doc-p {
  margin-top: 0.75rem;
  font-size: 0.9rem;
  line-height: 1.75;
  color: var(--muted-foreground);
}
.doc-table-wrap {
  margin-top: 0.75rem;
  overflow-x: auto;
}
.doc-meta {
  margin-top: 0.75rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.doc-method {
  border-radius: 0.25rem;
  background: color-mix(in oklch, var(--primary) 12%, transparent);
  padding: 0.1rem 0.5rem;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.03em;
  color: var(--primary);
}
.doc-url {
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 0.8rem;
  color: var(--foreground);
}
.doc-code {
  margin-top: 0.75rem;
  overflow-x: auto;
  border-radius: 0.5rem;
  border: 1px solid var(--border);
  background: oklch(0.18 0.02 264);
  padding: 1rem 1.25rem;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.7;
  color: oklch(0.9 0.02 264);
}
.doc-inline {
  border-radius: 0.25rem;
  background: var(--muted);
  padding: 0.1rem 0.4rem;
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 0.82em;
  color: var(--primary);
}
</style>
