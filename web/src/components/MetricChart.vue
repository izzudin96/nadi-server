<script setup lang="ts">
import * as echarts from 'echarts'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Point } from '@/api/client'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
  deviceId: string
  metricName: string
  unit?: string
}>()

const REFRESH_MS = 15000

const chartEl = ref<HTMLDivElement | null>(null)
const latest = ref<number | null>(null)
let chart: echarts.ECharts | null = null
let timer: number | undefined
let polling = false

onMounted(async () => {
  window.addEventListener('resize', resize)
  await load()
  timer = window.setInterval(poll, REFRESH_MS)
})
onBeforeUnmount(() => {
  if (timer !== undefined) window.clearInterval(timer)
  window.removeEventListener('resize', resize)
  chart?.dispose()
})

function resize() {
  chart?.resize()
}

async function poll() {
  if (polling) return
  polling = true
  try {
    await load()
  } finally {
    polling = false
  }
}

async function load() {
  const to = new Date()
  const from = new Date(to.getTime() - 60 * 60 * 1000)
  try {
    const res = await api.series(props.deviceId, props.metricName, from.toISOString(), to.toISOString())
    render(res.points)
  } catch {
    // transient error: keep the last render
  }
}

function render(points: Point[]) {
  if (!chartEl.value) return
  if (!chart) chart = echarts.init(chartEl.value)
  latest.value = points.length ? points[points.length - 1].value : null
  chart.setOption(
    {
      color: ['#4f46e5'],
      grid: { top: 8, right: 8, bottom: 24, left: 8 },
      tooltip: { trigger: 'axis' },
      xAxis: {
        type: 'time',
        axisLine: { lineStyle: { color: 'rgba(120,120,120,0.3)' } },
        axisLabel: { color: '#71717a', hideOverlap: true },
        splitLine: { show: false },
      },
      yAxis: {
        type: 'value',
        scale: true,
        splitLine: { lineStyle: { color: 'rgba(120,120,120,0.15)' } },
        axisLabel: { color: '#71717a' },
      },
      series: [
        {
          name: props.metricName,
          type: 'line',
          smooth: true,
          showSymbol: false,
          lineStyle: { width: 2 },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(79,70,229,0.25)' },
              { offset: 1, color: 'rgba(79,70,229,0)' },
            ]),
          },
          data: points.map((p) => [new Date(p.ts).getTime(), p.value]),
        },
      ],
    },
    true,
  )
}

function fmt(n: number): string {
  const a = Math.abs(n)
  if (a >= 1e9) return (n / 1e9).toFixed(2) + ' G'
  if (a >= 1e6) return (n / 1e6).toFixed(2) + ' M'
  if (a >= 1e3) return (n / 1e3).toFixed(2) + ' k'
  return n.toFixed(2)
}
</script>

<template>
  <Card size="sm">
    <CardHeader class="px-3">
      <CardTitle class="flex items-baseline justify-between gap-2">
        <span class="truncate font-medium">{{ metricName }}</span>
        <span v-if="latest !== null" class="text-muted-foreground whitespace-nowrap text-xs font-normal">
          {{ fmt(latest) }}<span v-if="unit"> {{ unit }}</span>
        </span>
      </CardTitle>
    </CardHeader>
    <CardContent class="px-3">
      <div ref="chartEl" class="h-40 w-full"></div>
    </CardContent>
  </Card>
</template>