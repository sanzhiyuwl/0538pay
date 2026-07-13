<script setup lang="ts">
import { cn } from '@/lib/utils'
import { cva, type VariantProps } from 'class-variance-authority'

const badgeVariants = cva(
  'inline-flex items-center rounded border px-2 py-[3px] text-xs font-normal leading-none transition-colors',
  {
    variants: {
      variant: {
        default: 'border-primary/25 bg-primary/[0.06] text-primary',
        success: 'border-success/30 bg-success/[0.07] text-success',
        warning: 'border-warning/35 bg-warning/[0.09] text-warning',
        destructive: 'border-destructive/25 bg-destructive/[0.06] text-destructive',
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
