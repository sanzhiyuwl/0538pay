<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowRight } from 'lucide-vue-next'
import { Button } from '@/components/ui'
import type { SiteContent } from '@/lib/mock/site-content'

defineProps<{ content: SiteContent; preview?: boolean }>()
const router = useRouter()
</script>

<template>
  <!-- ===== CTA（背景图 + 暗色遮罩 + 白色文字）===== -->
  <section class="relative overflow-hidden">
    <!-- 背景图 + 遮罩 -->
    <div class="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
      <img src="/assets/cta-bg.jpg" alt="" class="size-full object-cover" />
      <div class="absolute inset-0 bg-gradient-to-r from-[#0b1b3a]/90 to-[#0b1b3a]/70" />
    </div>
    <div class="relative z-10 mx-auto max-w-7xl px-4 py-24 text-center lg:px-6">
      <div v-reveal class="mx-auto max-w-2xl">
        <h2 class="text-3xl font-bold tracking-tight text-white sm:text-4xl">{{ content.cta.title }}</h2>
        <p class="mx-auto mt-3 max-w-lg text-white/75">{{ content.cta.subtitle }}</p>
        <div class="mt-8 flex flex-wrap justify-center gap-3">
          <Button size="lg" @click="router.push('/m/reg')">{{ content.cta.ctaPrimary }} <ArrowRight class="size-4" /></Button>
          <Button variant="outline" size="lg" class="border-white/40 bg-transparent text-white hover:bg-white/10 hover:text-white" @click="router.push('/docs')">{{ content.cta.ctaSecondary }}</Button>
        </div>
      </div>
    </div>
  </section>
</template>
