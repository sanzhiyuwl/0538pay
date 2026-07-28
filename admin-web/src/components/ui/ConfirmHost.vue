<script setup lang="ts">
/**
 * 全局确认弹窗渲染宿主。全站挂一个（App.vue），配合 useConfirm() 使用。
 * 视觉复用收单同款 Modal（居中 + 直角边框 + 底部右对齐按钮）。
 */
import { computed } from 'vue'
import { Modal, Button } from '@/components/ui'
import { useConfirm } from '@/composables/useConfirm'

const { state, settle } = useConfirm()
const open = computed(() => !!state.value)
</script>

<template>
  <Modal :model-value="open" :title="state?.title ?? ''" @update:model-value="(v) => { if (!v) settle(false) }">
    <p class="whitespace-pre-line text-sm text-muted-foreground">{{ state?.message }}</p>
    <template #footer>
      <Button variant="outline" size="sm" @click="settle(false)">{{ state?.cancelText }}</Button>
      <Button :variant="state?.danger ? 'destructive' : 'default'" size="sm" @click="settle(true)">
        {{ state?.confirmText }}
      </Button>
    </template>
  </Modal>
</template>
