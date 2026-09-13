<script setup lang="ts">
import { AlertCircle, ChevronLeft, LineChart } from 'lucide-vue-next'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { LatestMetric } from '@/api/client'
import MetricChart from '@/components/MetricChart.vue'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

const RANGES = [
  { label: '1h', seconds: 3600 },
  { label: '3h', seconds: 10800 },
  { label: '6h', seconds: 21600 },
  { label: '12h', seconds: 43200 },
  { label: '1d', seconds: 86400 },
  { label: '3d', seconds: 259200 },
  { label: '7d', seconds: 604800 },
  { label: '30d', seconds: 2592000 },
]

const route = useRoute()
const router = useRouter()
const deviceID = route.params.id as string

// Each chart polls its own series; this slower poll just picks up newly
// reported metric names so their graphs appear without a reload.
const LIST_REFRESH_MS = 30000

const metrics = ref<LatestMetric[]>([])
const rangeSeconds = ref(RANGES[0].seconds)
const loading = ref(true)
const error = ref('')
let timer: number | undefined

onMounted(async () => {
  await loadLatest()
  timer = window.setInterval(loadLatest, LIST_REFRESH_MS)
})
onBeforeUnmount(() => {
  if (timer !== undefined) window.clearInterval(timer)
})

async function loadLatest() {
  try {
    const res = await api.latest(deviceID)
    metrics.value = res.metrics
    error.value = ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load metrics'
  } finally {
    loading.value = false
  }
}

function rangeLabel(): string {
  const r = RANGES.find((r) => r.seconds === rangeSeconds.value)
  return r ? r.label : `${rangeSeconds.value / 3600}h`
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
      <p class="text-muted-foreground mt-1 text-sm">Live metrics, showing the last {{ rangeLabel() }}.</p>
    </div>

    <div class="mb-6 flex flex-wrap items-center gap-1">
      <Button
        v-for="r in RANGES"
        :key="r.seconds"
        size="sm"
        :variant="rangeSeconds === r.seconds ? 'default' : 'outline'"
        @click="rangeSeconds = r.seconds"
      >
        {{ r.label }}
      </Button>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">
      <AlertCircle />
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <div v-if="loading" class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <Skeleton v-for="i in 9" :key="i" class="h-60 rounded-xl" />
    </div>

    <div v-else-if="metrics.length" class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <MetricChart
        v-for="m in metrics"
        :key="m.name"
        :device-id="deviceID"
        :metric-name="m.name"
        :unit="m.unit"
        :range-seconds="rangeSeconds"
      />
    </div>

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