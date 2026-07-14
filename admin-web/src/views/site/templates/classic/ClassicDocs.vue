<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { BookOpen } from 'lucide-vue-next'

// 文档目录（锚点滚动）
const toc = [
  { id: 'intro', title: '接入概述' },
  { id: 'sign', title: '签名规则' },
  { id: 'create', title: '统一下单' },
  { id: 'notify', title: '异步通知' },
  { id: 'refund', title: '申请退款' },
  { id: 'query', title: '订单查询' },
  { id: 'errcode', title: '错误码' },
]
const active = ref('intro')

function scrollTo(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// 滚动高亮当前章节
let observer: IntersectionObserver | null = null
onMounted(() => {
  observer = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (e.isIntersecting) active.value = e.target.id
      }
    },
    { rootMargin: '-20% 0px -70% 0px' },
  )
  toc.forEach((t) => {
    const el = document.getElementById(t.id)
    if (el) observer!.observe(el)
  })
})
onUnmounted(() => observer?.disconnect())

// 下单请求参数
const createParams = [
  { name: 'pid', type: 'int', req: '是', desc: '商户 ID' },
  { name: 'type', type: 'string', req: '是', desc: '支付方式：alipay / wxpay / qqpay / bank' },
  { name: 'out_trade_no', type: 'string', req: '是', desc: '商户订单号，同商户下唯一' },
  { name: 'notify_url', type: 'string', req: '是', desc: '异步通知地址（服务器回调）' },
  { name: 'return_url', type: 'string', req: '是', desc: '同步跳转地址（支付完成返回）' },
  { name: 'name', type: 'string', req: '是', desc: '商品名称' },
  { name: 'money', type: 'string', req: '是', desc: '金额，单位元，保留两位小数' },
  { name: 'sign', type: 'string', req: '是', desc: '签名字符串，见签名规则' },
  { name: 'sign_type', type: 'string', req: '是', desc: '签名类型，固定 MD5 或 RSA' },
]

// 错误码
const errcodes = [
  { code: '1', text: '成功' },
  { code: '-1', text: '签名校验失败' },
  { code: '-2', text: '商户不存在或已禁用' },
  { code: '-3', text: '缺少必要参数' },
  { code: '-4', text: '订单号重复' },
  { code: '-5', text: '金额不合法' },
  { code: '-6', text: '无可用支付通道' },
  { code: '-7', text: '风控拦截' },
]

const signCode = `// 1. 参数按键名 ASCII 升序排序，拼接为 key=value&key=value
// 2. 末尾拼接商户密钥（MD5）或用商户私钥签名（RSA）
$params = [
  'pid'          => 1001,
  'type'         => 'alipay',
  'out_trade_no' => '20260714001',
  'name'         => 'VIP会员',
  'money'        => '9.90',
];
ksort($params);
$str = urldecode(http_build_query($params));
$sign = md5($str . $merchantKey);   // MD5 签名`

const createCode = `POST https://0538pay.com/mapi.php
Content-Type: application/x-www-form-urlencoded

pid=1001&type=alipay&out_trade_no=20260714001
&notify_url=https://your.site/notify
&return_url=https://your.site/return
&name=VIP会员&money=9.90
&sign=xxxxxxxx&sign_type=MD5

// 返回
{
  "code": 1,
  "trade_no": "20260714120000123456",
  "payurl": "https://0538pay.com/pay/alipay/xxxx"
}`

const notifyCode = `// 平台向 notify_url 以 GET/POST 发送：
trade_no=20260714120000123456
&out_trade_no=20260714001
&type=alipay&name=VIP会员&money=9.90
&trade_status=TRADE_SUCCESS
&sign=xxxxxxxx&sign_type=MD5

// 商户验签通过后，务必原样输出 success（否则平台会重试）
echo 'success';`
</script>

<template>
  <div class="site-surface min-h-[70vh] border-b border-border">
    <div class="mx-auto flex max-w-[1600px] gap-12 px-6 py-12 lg:px-12 xl:gap-16">
      <!-- 左侧目录 -->
      <aside class="hidden w-72 shrink-0 lg:block">
        <div class="sticky top-24 rounded-2xl border border-border bg-background p-4 shadow-sm">
          <div class="flex items-center gap-2 px-2 pb-3">
            <div class="flex size-8 items-center justify-center rounded-lg bg-primary/[0.08] text-primary">
              <BookOpen class="size-[18px]" />
            </div>
            <div class="text-sm font-semibold">开发者文档</div>
          </div>
          <div class="border-t border-border/60" />
          <nav class="mt-3 space-y-1">
            <button
              v-for="(t, i) in toc"
              :key="t.id"
              class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-sm transition-colors"
              :class="active === t.id
                ? 'bg-primary font-medium text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
              @click="scrollTo(t.id)"
            >
              <span
                class="flex size-5 shrink-0 items-center justify-center rounded text-[11px] font-medium tabular-nums"
                :class="active === t.id ? 'bg-white/20 text-primary-foreground' : 'bg-muted text-muted-foreground'"
              >{{ i + 1 }}</span>
              {{ t.title }}
            </button>
          </nav>
        </div>
      </aside>

      <!-- 右侧内容 -->
      <div class="min-w-0 flex-1 space-y-14">
        <header>
          <h1 class="text-3xl font-bold tracking-tight">接入文档</h1>
          <p class="mt-2 text-muted-foreground">最快 1 天完成对接。以下为 V1（MD5）接口示例，V2（RSA）签名方式见签名规则。</p>
        </header>

        <!-- 接入概述 -->
        <section id="intro" class="scroll-mt-24">
          <h2 class="text-xl font-semibold">接入概述</h2>
          <p class="mt-3 leading-relaxed text-muted-foreground">
            0538Pay 提供聚合支付 API，商户注册并实名后，在「API 信息」页获取商户 ID 与密钥即可对接。
            核心流程：<b class="text-foreground">下单 → 跳转支付 → 异步通知入账 → （可选）退款/查询</b>。
          </p>
          <div class="mt-4 grid gap-3 sm:grid-cols-3">
            <div v-for="s in ['1. 获取商户 ID 与密钥', '2. 按签名规则构造请求', '3. 调用下单接口并跳转']" :key="s" class="rounded-lg border border-border bg-background p-4 text-sm">{{ s }}</div>
          </div>
        </section>

        <!-- 签名规则 -->
        <section id="sign" class="scroll-mt-24">
          <h2 class="text-xl font-semibold">签名规则</h2>
          <p class="mt-3 leading-relaxed text-muted-foreground">
            参数按键名 ASCII 升序排序后拼接，MD5 模式末尾追加商户密钥取 md5；RSA 模式用商户私钥 SHA256withRSA 签名。
          </p>
          <pre class="doc-code"><code>{{ signCode }}</code></pre>
        </section>

        <!-- 统一下单 -->
        <section id="create" class="scroll-mt-24">
          <h2 class="text-xl font-semibold">统一下单</h2>
          <p class="mt-3 leading-relaxed text-muted-foreground">请求参数：</p>
          <div class="mt-3 overflow-x-auto">
            <table class="tbl w-full">
              <thead>
                <tr><th class="w-[18%]">参数</th><th class="w-[14%]">类型</th><th class="w-[10%]">必填</th><th>说明</th></tr>
              </thead>
              <tbody>
                <tr v-for="p in createParams" :key="p.name">
                  <td class="font-mono text-[13px] text-primary">{{ p.name }}</td>
                  <td class="text-muted-foreground">{{ p.type }}</td>
                  <td>{{ p.req }}</td>
                  <td class="text-muted-foreground">{{ p.desc }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <pre class="doc-code mt-4"><code>{{ createCode }}</code></pre>
        </section>

        <!-- 异步通知 -->
        <section id="notify" class="scroll-mt-24">
          <h2 class="text-xl font-semibold">异步通知</h2>
          <p class="mt-3 leading-relaxed text-muted-foreground">
            支付成功后平台向 <code class="doc-inline">notify_url</code> 推送结果，商户验签通过后需原样返回
            <code class="doc-inline">success</code>，否则平台按 1/3/20 分钟、1/2 小时重试。
          </p>
          <pre class="doc-code"><code>{{ notifyCode }}</code></pre>
        </section>

        <!-- 申请退款 -->
        <section id="refund" class="scroll-mt-24">
          <h2 class="text-xl font-semibold">申请退款</h2>
          <p class="mt-3 leading-relaxed text-muted-foreground">
            调用 <code class="doc-inline">api.php?act=refund</code>，传商户 ID、订单号、退款金额与签名。需商户开启退款 API 权限。
            支持部分退款，退款金额不可超过实付金额。
          </p>
        </section>

        <!-- 订单查询 -->
        <section id="query" class="scroll-mt-24">
          <h2 class="text-xl font-semibold">订单查询</h2>
          <p class="mt-3 leading-relaxed text-muted-foreground">
            调用 <code class="doc-inline">api.php?act=order</code> 按系统订单号或商户订单号查询订单状态，
            返回订单金额、支付状态、完成时间等。建议以异步通知为准，查询作为补偿手段。
          </p>
        </section>

        <!-- 错误码 -->
        <section id="errcode" class="scroll-mt-24">
          <h2 class="text-xl font-semibold">错误码</h2>
          <div class="mt-3 overflow-x-auto">
            <table class="tbl w-full max-w-md">
              <thead><tr><th class="w-[30%]">code</th><th>说明</th></tr></thead>
              <tbody>
                <tr v-for="e in errcodes" :key="e.code">
                  <td class="font-mono" :class="e.code === '1' ? 'text-success' : 'text-destructive'">{{ e.code }}</td>
                  <td class="text-muted-foreground">{{ e.text }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
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
  font-size: 0.85em;
  color: var(--primary);
}
</style>
