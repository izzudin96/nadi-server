<script setup lang="ts">
import * as echarts from 'echarts'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api/client'
import type { LatestMetric, Point } from '../api/client'

const route = useRoute()
const deviceID = route.params.id as string

const metrics = ref<LatestMetric[]>([])
const selected = ref<string | null>(null)
const error = ref('')
const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null

onMounted(loadLatest)
onBeforeUnmount(() => chart?.dispose())

async function loadLatest() {
  try {
    const res = await api.latest(deviceID)
    metrics.value = res.metrics
    if (metrics.value.length && !selected.value) {
      select(metrics.value[0].name)
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load metrics'
  }
}

async function select(name: string) {
  selected.value = name
  const to = new Date()
  const from = new Date(to.getTime() - 60 * 60 * 1000)
  try {
    const res = await api.series(deviceID, name, from.toISOString(), to.toISOString())
    renderChart(name, res.points)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load series'
  }
}

function renderChart(name: string, points: Point[]) {
  if (!chartEl.value) return
  if (!chart) chart = echarts.init(chartEl.value)
  chart.setOption({
    title: { text: name, textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'time' },
    yAxis: { type: 'value', scale: true },
    series: [
      {
        name,
        type: 'line',
        showSymbol: false,
        data: points.map((p) => [new Date(p.ts).getTime(), p.value]),
      },
    ],
  })
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
  <div class="mx-auto max-w-6xl px-4 py-6">
    <router-link to="/" class="mb-4 inline-block text-sm text-blue-600 hover:underline">
      ← All devices
    </router-link>
    <h1 class="mb-6 text-2xl font-semibold">{{ deviceID }}</h1>

    <p v-if="error" class="text-red-600">{{ error }}</p>

    <div v-if="metrics.length" class="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
      <button
        v-for="m in metrics"
        :key="m.name"
        class="rounded-lg border p-4 text-left transition"
        :class="selected === m.name ? 'border-blue-500 bg-blue-50' : 'border-gray-200 bg-white'"
        @click="select(m.name)"
      >
        <div class="truncate text-xs text-gray-500">{{ m.name }}</div>
        <div class="mt-1 text-xl font-semibold">
          {{ fmt(m) }}<span class="ml-1 text-sm font-normal text-gray-400">{{ m.unit }}</span>
        </div>
      </button>
    </div>

    <div v-if="metrics.length" class="rounded-lg bg-white p-4 shadow">
      <div ref="chartEl" class="h-96 w-full"></div>
    </div>
    <p v-else class="text-gray-500">No metrics yet.</p>
  </div>
</template>
