<script setup lang="ts">
import { AlertCircle, MonitorSmartphone, RefreshCw } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { Device } from '@/api/client'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

const router = useRouter()
const devices = ref<Device[]>([])
const error = ref('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.devices()
    devices.value = res.devices
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load devices'
  } finally {
    loading.value = false
  }
}

function fmt(ts: string | null): string {
  if (!ts) return 'Never'
  return new Date(ts).toLocaleString()
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8">
    <div class="mb-6 flex items-end justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Devices</h1>
        <p class="text-muted-foreground mt-1 text-sm">
          Monitor every agent reporting to your account.
        </p>
      </div>
      <Button variant="outline" :disabled="loading" @click="load">
        <RefreshCw :class="cn(loading && 'animate-spin')" />
        Refresh
      </Button>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">
      <AlertCircle />
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <Card class="py-0">
      <Table>
        <TableHeader>
          <TableRow class="hover:bg-transparent">
            <TableHead class="px-4">Status</TableHead>
            <TableHead class="px-4">Device</TableHead>
            <TableHead class="px-4">Hostname</TableHead>
            <TableHead class="px-4">OS</TableHead>
            <TableHead class="px-4">Last seen</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="loading">
            <TableRow v-for="i in 5" :key="i" class="hover:bg-transparent">
              <TableCell v-for="c in 5" :key="c" class="px-4 py-3">
                <Skeleton class="h-4 w-full max-w-32" />
              </TableCell>
            </TableRow>
          </template>

          <TableRow v-else-if="devices.length === 0" class="hover:bg-transparent">
            <TableCell colspan="5" class="px-4 py-16">
              <div class="flex flex-col items-center gap-2 text-center">
                <span class="bg-muted text-muted-foreground grid size-10 place-items-center rounded-full">
                  <MonitorSmartphone class="size-5" />
                </span>
                <p class="font-medium">No devices yet</p>
                <p class="text-muted-foreground text-sm">
                  Install the Nadi agent to start reporting metrics.
                </p>
              </div>
            </TableCell>
          </TableRow>

          <template v-else>
            <TableRow
              v-for="d in devices"
              :key="d.device_id"
              class="cursor-pointer"
              @click="router.push(`/devices/${d.device_id}`)"
            >
              <TableCell class="px-4 py-3">
                <Badge
                  variant="outline"
                  :class="
                    d.online
                      ? 'border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400'
                      : 'text-muted-foreground border-transparent bg-muted'
                  "
                >
                  <span
                    :class="cn('size-1.5 rounded-full', d.online ? 'bg-emerald-500' : 'bg-muted-foreground/50')"
                  />
                  {{ d.online ? 'Online' : 'Offline' }}
                </Badge>
              </TableCell>
              <TableCell class="px-4 py-3 font-medium">{{ d.device_id }}</TableCell>
              <TableCell class="text-muted-foreground px-4 py-3">{{ d.hostname || '—' }}</TableCell>
              <TableCell class="text-muted-foreground px-4 py-3">{{ d.os || '—' }}</TableCell>
              <TableCell class="text-muted-foreground px-4 py-3">{{ fmt(d.last_seen_at) }}</TableCell>
            </TableRow>
          </template>
        </TableBody>
      </Table>
    </Card>
  </div>
</template>
