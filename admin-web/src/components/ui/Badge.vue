<script setup lang="ts">
import { cn } from '@/lib/utils'
import { cva, type VariantProps } from 'class-variance-authority'

/**
 * 状态标签。样式 1:1 复刻 wepay /admin/users（Element UI 配色的淡底描边 badge）：
 * font-weight 500 / 12px / padding 0 8px / line-height 20px / 圆角 4px / 1px 描边。
 * 淡底 + 同色系字 + 更浅同色描边。颜色按语义。
 */
const badgeVariants = cva(
  'inline-flex items-center rounded border px-2 text-xs font-medium leading-5 transition-colors',
  {
    variants: {
      variant: {
        // default 用本项目主色（蓝）的淡底描边
        default: 'border-primary/25 bg-primary/[0.08] text-primary',
        // 以下 success/warning/destructive 直接采用 Element UI 精确色值，与 wepay 一致
        success: 'border-[#e1f3d8] bg-[#f0f9eb] text-[#67C23A]',
        warning: 'border-[#faecd8] bg-[#fdf6ec] text-[#E6A23C]',
        destructive: 'border-[#fde2e2] bg-[#fef0f0] text-[#F56C6C]',
        muted: 'border-border bg-muted/50 text-muted-foreground',
        outline: 'border-border text-foreground',
      },
    },
    defaultVariants: { variant: 'default' },
  },
)

type Variant = VariantProps<typeof badgeVariants>['variant']
const props = defineProps<{ variant?: Variant; class?: string }>()
</script>

<template>
  <span :class="cn(badgeVariants({ variant: props.variant }), props.class)">
    <slot />
  </span>
</template>
