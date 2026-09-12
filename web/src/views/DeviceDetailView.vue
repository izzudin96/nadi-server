<script setup lang="ts">
import * as echarts from 'echarts'
import { AlertCircle, ChevronLeft, LineChart } from 'lucide-vue-next'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { LatestMetric, Point } from '@/api/client'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const deviceID = route.params.id as string

const REFRESH_MS = 15000

const metrics = ref<LatestMetric[]>([])
const selected = ref<string | null>(null)
const loading = ref(true)
const error = ref('')
const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let timer: number | undefined
let polling = false

onMounted(async () => {
  await loadLatest()
  window.addEventListener('resize', resizeChart)
  timer = window.setInterval(poll, REFRESH_MS)
})
onBeforeUnmount(() => {
  if (timer !== undefined) window.clearInterval(timer)
  window.removeEventListener('resize', resizeChart)
  chart?.dispose()
})

function resizeChart() {
  chart?.resize()
}

// poll refreshes the cards and chart in the background; it never flashes the
// skeleton or surfaces transient errors, so the page stays stable while live.
async function poll() {
  if (polling) return
  polling = true
  try {
    await loadLatest(true)
  } finally {
    polling = false
  }
}

async function loadLatest(silent = false) {
  if (!silent) loading.value = true
  try {
    const res = await api.latest(deviceID)
    metrics.value = res.metrics
    error.value = ''
    if (!selected.value) {
      if (metrics.value.length) selected.value = metrics.value[0].name
    }
    await loadSeries()
  } catch (e) {
    if (!silent) error.value = e instanceof Error ? e.message : 'Failed to load metrics'
  } finally {
    if (!silent) loading.value = false
  }
}

async function select(name: string) {
  selected.value = name
  await loadSeries()
}

async function loadSeries() {
  if (!selected.value) return
  const to = new Date()
  const from = new Date(to.getTime() - 60 * 60 * 1000)
  try {
    const res = await api.series(deviceID, selected.value, from.toISOString(), to.toISOString())
    renderChart(selected.value, res.points)
    error.value = ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load series'
  }
}

function renderChart(name: string, points: Point[]) {
  if (!chartEl.value) return
  if (!chart) chart = echarts.init(chartEl.value)
  chart.setOption(
    {
      color: ['#4f46e5'],
      grid: { top: 16, right: 16, bottom: 32, left: 48 },
      tooltip: { trigger: 'axis' },
      xAxis: {
        type: 'time',
        axisLine: { lineStyle: { color: 'rgba(120,120,120,0.3)' } },
        axisLabel: { color: '#71717a' },
      },
      yAxis: {
        type: 'value',
        scale: true,
        splitLine: { lineStyle: { color: 'rgba(120,120,120,0.15)' } },
        axisLabel: { color: '#71717a' },
      },
      series: [
        {
          name,
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

function fmt(m: LatestMetric): string {
  const a = Math.abs(m.value)
  if (a >= 1e9) return (m.value / 1e9).toFixed(2) + ' G'
  if (a >= 1e6) return (m.value / 1e6).toFixed(2) + ' M'
  if (a >= 1e3) return (m.value / 1e3).toFixed(2) + ' k'
  return m.value.toFixed(2)
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8">
    <Button variant="ghost" size="sm" class="-ml-2 mb-4" @click="router.push('/')">
      <ChevronLeft />
      All devices
    </Button>

    <div class="mb-6">
      <h1 class="text-2xl font-semibold tracking-tight">{{ deviceID }}</h1>
      <p class="text-muted-foreground mt-1 text-sm">Metrics collected over the last hour.</p>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">
      <AlertCircle />
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <div v-if="loading" class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
      <Skeleton v-for="i in 8" :key="i" class="h-24 rounded-xl" />
    </div>

    <template v-else-if="metrics.length">
      <Card class="mb-6">
        <CardHeader>
          <CardTitle>{{ selected }}</CardTitle>
          <CardDescription>Last 60 minutes</CardDescription>
        </CardHeader>
        <CardContent>
          <div ref="chartEl" class="h-96 w-full"></div>
        </CardContent>
      </Card>

      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <button
          v-for="m in metrics"
          :key="m.name"
          type="button"
          :class="
            cn(
              'bg-card text-card-foreground rounded-xl p-4 text-left ring-1 transition-all outline-none',
              'hover:ring-foreground/20 focus-visible:ring-2 focus-visible:ring-ring',
              selected === m.name ? 'ring-2 ring-primary' : 'ring-foreground/10',
            )
          "
          @click="select(m.name)"
        >
          <div class="text-muted-foreground truncate text-xs font-medium">{{ m.name }}</div>
          <div class="mt-2 text-2xl font-semibold tracking-tight">
            {{ fmt(m) }}<span class="text-muted-foreground ml-1 text-sm font-normal">{{ m.unit }}</span>
          </div>
        </button>
      </div>
    </template>

    <Card v-else class="py-16">
      <CardContent class="flex flex-col items-center gap-2 text-center">
        <span class="bg-muted text-muted-foreground grid size-10 place-items-center rounded-full">
          <LineChart class="size-5" />
        </span>
        <p class="font-medium">No metrics yet</p>
        <p class="text-muted-foreground text-sm">This device hasn't reported any metrics.</p>
      </CardContent>
    </Card>
  </div>
</template>
