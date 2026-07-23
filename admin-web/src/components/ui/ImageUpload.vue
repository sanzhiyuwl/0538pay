<script setup lang="ts">
/**
 * 图片上传字段。点击上传本地图片（走 /api/admin/upload/image），或手动填写图片地址。
 * v-model 绑定图片 URL 字符串（相对路径如 /uploads/article/xxx.png，或外链）。
 *
 * <ImageUpload v-model="form.cover" dir="cover" />
 */
import { ref } from 'vue'
import { ImagePlus, Loader2, RefreshCw, Trash2 } from 'lucide-vue-next'
import { uploadImage } from '@/lib/api/upload'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/api/client'

const props = withDefaults(
  defineProps<{
    modelValue: string
    dir?: 'article' | 'cover' | 'category'
  }>(),
  { dir: 'cover' },
)
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const toast = useToast()
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

function pick() {
  fileInput.value?.click()
}

async function onFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // 清空以便重选同名文件
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.error('请选择图片文件')
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    toast.error('图片过大，单张不超过 10MB')
    return
  }
  uploading.value = true
  try {
    const url = await uploadImage(file, props.dir)
    emit('update:modelValue', url)
    toast.success('图片已上传')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '上传失败')
  } finally {
    uploading.value = false
  }
}

function clear() {
  emit('update:modelValue', '')
}
</script>

<template>
  <div class="flex items-start gap-3">
    <!-- 上传方框：空态点击上传；有图时悬浮显示更换/删除 -->
    <div
      class="group relative flex size-28 shrink-0 items-center justify-center overflow-hidden rounded-md border border-dashed border-input bg-muted/30 transition-colors hover:border-primary/50"
    >
      <img v-if="modelValue" :src="modelValue" alt="封面预览" class="size-full object-cover" />
      <!-- 空态：整块可点击上传 -->
      <button
        v-else
        type="button"
        class="flex size-full flex-col items-center justify-center gap-1.5 text-muted-foreground/60 transition-colors hover:text-primary"
        :disabled="uploading"
        @click="pick"
      >
        <ImagePlus class="size-6" />
        <span class="text-xs">上传封面</span>
      </button>

      <!-- 上传中遮罩 -->
      <div v-if="uploading" class="absolute inset-0 flex items-center justify-center bg-background/70">
        <Loader2 class="size-5 animate-spin text-primary" />
      </div>

      <!-- 有图悬浮操作：更换 / 删除 -->
      <div
        v-if="modelValue && !uploading"
        class="absolute inset-0 flex items-center justify-center gap-3 bg-black/50 opacity-0 transition-opacity group-hover:opacity-100"
      >
        <button type="button" title="更换图片" class="text-white/90 transition-colors hover:text-white" @click="pick">
          <RefreshCw class="size-5" />
        </button>
        <button type="button" title="删除图片" class="text-white/90 transition-colors hover:text-destructive" @click="clear">
          <Trash2 class="size-5" />
        </button>
      </div>
    </div>

    <!-- 说明文案 -->
    <p class="pt-1 text-xs leading-relaxed text-muted-foreground/70">
      点击方框上传本地图片。<br />
      支持 jpg / png / gif / webp，单张 ≤ 10MB。
    </p>

    <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="onFile" />
  </div>
</template>
