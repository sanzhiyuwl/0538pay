<script setup lang="ts">
import { Panel } from '@/components/ui'
import { Zap, Clock, Percent, Landmark } from 'lucide-vue-next'

const sitename = '0538Pay'
</script>

<template>
  <div class="space-y-2.5">
    <Panel title="使用说明" subtitle="平台交易规则与到账说明">
      <div class="max-w-3xl space-y-6">
        <!-- 即时到账 -->
        <section>
          <h4 class="flex items-center gap-2 text-sm font-semibold"><Zap class="size-4 text-primary" />一、交易即时到账</h4>
          <p class="mt-2 text-sm leading-relaxed text-muted-foreground">
            {{ sitename }} 支持支付宝、微信、财付通、QQ钱包等多种支付方式，买家付款后款项实时到账至您的商户余额，无需等待。
          </p>
        </section>

        <!-- 提现方案 -->
        <section class="border-t border-border/60 pt-5">
          <h4 class="flex items-center gap-2 text-sm font-semibold"><Clock class="size-4 text-primary" />二、T+1 提现方案</h4>
          <p class="mt-2 text-sm leading-relaxed text-muted-foreground">
            商户余额支持手动申请提现，默认 T+1 到账（当日交易次日可提）。提现申请提交后由系统或人工审核后下发到您的收款账户。
          </p>
        </section>

        <!-- 提现费率 -->
        <section class="border-t border-border/60 pt-5">
          <h4 class="flex items-center gap-2 text-sm font-semibold"><Percent class="size-4 text-primary" />三、提现费率</h4>
          <div class="mt-3 overflow-x-auto">
            <table class="tbl w-full">
              <thead>
                <tr>
                  <th>项目</th>
                  <th>规则</th>
                </tr>
              </thead>
              <tbody>
                <tr><td>起提金额</td><td>单笔 ≥ 10 元</td></tr>
                <tr><td>提现费率</td><td>0.5%，最高 25 元</td></tr>
                <tr><td>最低手续费</td><td>不足 1 元按 1 元收取</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 结算方式 -->
        <section class="border-t border-border/60 pt-5">
          <h4 class="flex items-center gap-2 text-sm font-semibold"><Landmark class="size-4 text-primary" />四、结算方式</h4>
          <ul class="mt-2 space-y-1.5 text-sm leading-relaxed text-muted-foreground">
            <li>· 小额提现：官方企业支付宝 → 您的个人支付宝，实时到账。</li>
            <li>· 大额提现：官方对公账户 → 您的个人银行卡，1~2 个工作日到账。</li>
            <li>· 具体结算方式与限额以平台风控策略为准，如有疑问请联系客服。</li>
          </ul>
        </section>
      </div>
    </Panel>
  </div>
</template>
