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

const route = useRoute()
const router = useRouter()
const deviceID = route.params.id as string

// Charts poll their own series every 15s; this slower poll just picks up newly
// reported metric names so their graphs appear without a reload.
const LIST_REFRESH_MS = 30000

const metrics = ref<LatestMetric[]>([])
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
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8">
    <Button variant="ghost" size="sm" class="-ml-2 mb-4" @click="router.push('/')">
      <ChevronLeft />
      All devices
    </Button>

    <div class="mb-6">
      <h1 class="text-2xl font-semibold tracking-tight">{{ deviceID }}</h1>
      <p class="text-muted-foreground mt-1 text-sm">Live metrics collected over the last hour.</p>
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