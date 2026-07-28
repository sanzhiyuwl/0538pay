<script setup lang="ts">
import { RouterView } from 'vue-router'
import { ToastHost, ConfirmHost } from '@/components/ui'
</script>

<template>
  <RouterView />
  <ToastHost />
  <ConfirmHost />
</template>
