<script setup lang="ts">
import { computed } from 'vue'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  LineElement,
  PointElement,
  CategoryScale,
  LinearScale,
  Filler,
} from 'chart.js'
import { useThemeStore } from '@/stores/theme'

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  LineElement,
  PointElement,
  CategoryScale,
  LinearScale,
  Filler,
)

const props = defineProps<{
  labels: string[]
  series: { name: string; data: number[]; color: string }[]
}>()

const theme = useThemeStore()

const chartData = computed(() => ({
  labels: props.labels,
  datasets: props.series.map((s) => ({
    label: s.name,
    data: s.data,
    borderColor: s.color,
    backgroundColor: s.color + '20',
    fill: true,
    tension: 0.4,
    borderWidth: 2,
    pointRadius: 0,
    pointHoverRadius: 5,
    pointHoverBackgroundColor: s.color,
    pointHoverBorderColor: '#fff',
    pointHoverBorderWidth: 2,
  })),
}))

const chartOptions = computed(() => {
  const grid = theme.isDark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.05)'
  const tick = theme.isDark ? 'rgba(255,255,255,0.65)' : 'rgba(0,0,0,0.7)'
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    plugins: {
      legend: {
        position: 'bottom' as const,
        labels: {
          usePointStyle: true,
          pointStyle: 'circle',
          boxWidth: 6,
          boxHeight: 6,
          padding: 24,
          color: tick,
          font: { size: 12 },
          // 去掉隐藏系列时的默认删除线，改用降低透明度表示“已关闭”
          generateLabels: (chart: any) => {
            const ds = chart.data.datasets as { label?: string; borderColor?: string }[]
            return ds.map((d, i) => {
              const hidden = !chart.isDatasetVisible(i)
              return {
                text: d.label ?? '',
                fillStyle: d.borderColor,
                strokeStyle: d.borderColor,
                pointStyle: 'circle',
                hidden,
                fontColor: hidden
                  ? theme.isDark
                    ? 'rgba(255,255,255,0.3)'
                    : 'rgba(0,0,0,0.3)'
                  : tick,
                datasetIndex: i,
                lineWidth: 0,
                textDecoration: '',
              }
            })
          },
        },
      },
      tooltip: {
        backgroundColor: theme.isDark ? '#1f2130' : '#fff',
        titleColor: theme.isDark ? '#fff' : '#111',
        bodyColor: tick,
        borderColor: grid,
        borderWidth: 1,
        padding: 12,
        cornerRadius: 8,
        usePointStyle: true,
        callbacks: {
          label: (ctx: { dataset: { label?: string }; parsed: { y: number } }) =>
            ` ${ctx.dataset.label}: ¥${ctx.parsed.y.toLocaleString()}`,
        },
      },
    },
    scales: {
      x: {
        grid: { display: false },
        border: { display: false },
        ticks: { color: tick, font: { size: 12 } },
      },
      y: {
        grid: { color: grid },
        border: { display: false },
        ticks: {
          color: tick,
          font: { size: 12 },
          callback: (v: number | string) =>
            '¥' + Number(v).toLocaleString(),
        },
      },
    },
  }
})
</script>

<template>
  <div class="h-[300px]">
    <Line :data="chartData" :options="chartOptions as any" />
  </div>
</template>
