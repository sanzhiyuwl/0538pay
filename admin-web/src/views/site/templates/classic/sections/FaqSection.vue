<script setup lang="ts">
import { ref } from 'vue'
import { ChevronDown } from 'lucide-vue-next'
import type { SiteContent } from '@/lib/mock/site-content'

defineProps<{ content: SiteContent; preview?: boolean }>()

// 手风琴：默认展开第一条，点击切换
const openIndex = ref(0)
function toggle(i: number) {
  openIndex.value = openIndex.value === i ? -1 : i
}
</script>

<template>
  <!-- ===== 常见问题（居中标题 + 手风琴问答）===== -->
  <section id="faqs" class="scroll-mt-20 bg-content">
    <div class="mx-auto max-w-3xl px-4 py-24 lg:px-6">
      <div v-reveal class="mx-auto max-w-2xl text-center">
        <h2 class="text-3xl font-bold tracking-tight">{{ content.faqTitle }}</h2>
        <p class="mt-3 text-muted-foreground">{{ content.faqSubtitle }}</p>
      </div>
      <div v-reveal class="mt-12 space-y-3">
        <div
          v-for="(f, i) in content.faqs"
          :key="f.q"
          class="overflow-hidden bg-background shadow-[0_1px_3px_rgba(0,0,0,0.04)] ring-1 ring-border"
        >
          <button
            type="button"
            class="flex w-full items-center gap-4 px-6 py-5 text-left"
            @click="toggle(i)"
          >
            <span class="flex-1 text-base font-semibold">{{ f.q }}</span>
            <ChevronDown
              class="size-5 shrink-0 text-muted-foreground transition-transform"
              :class="openIndex === i ? 'rotate-180 text-primary' : ''"
            />
          </button>
          <div v-show="openIndex === i" class="px-6 pb-5 text-sm leading-relaxed text-muted-foreground">
            {{ f.a }}
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
