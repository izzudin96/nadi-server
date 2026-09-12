<script setup lang="ts">
import {
  AlertCircle,
  Check,
  Copy,
  KeyRound,
  MonitorSmartphone,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Trash2,
  X,
} from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { Device, DeviceKey } from '@/api/client'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
const busy = ref(false)

const showCreate = ref(false)
const newDeviceId = ref('')
const createError = ref('')

const issued = ref<DeviceKey | null>(null)
const copied = ref(false)

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

function openCreate() {
  newDeviceId.value = ''
  createError.value = ''
  showCreate.value = true
}

async function createDevice() {
  createError.value = ''
  busy.value = true
  try {
    issued.value = await api.createDevice(newDeviceId.value.trim())
    showCreate.value = false
    await load()
  } catch (e) {
    createError.value = e instanceof Error ? e.message : 'Failed to create device'
  } finally {
    busy.value = false
  }
}

async function rotate(d: Device) {
  if (!window.confirm(`Rotate the API key for ${d.device_id}? The old key stops working immediately.`)) {
    return
  }
  busy.value = true
  try {
    issued.value = await api.rotateDevice(d.device_id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to rotate key'
  } finally {
    busy.value = false
  }
}

async function remove(d: Device) {
  if (!window.confirm(`Delete ${d.device_id} and all of its metrics? This cannot be undone.`)) {
    return
  }
  busy.value = true
  try {
    await api.deleteDevice(d.device_id)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete device'
  } finally {
    busy.value = false
  }
}

async function copyKey() {
  if (!issued.value) return
  try {
    await navigator.clipboard.writeText(issued.value.api_key)
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    copied.value = false
  }
}

function closeIssued() {
  issued.value = null
  copied.value = false
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
      <div class="flex items-center gap-2">
        <Button variant="outline" :disabled="loading" @click="load">
          <RefreshCw :class="cn(loading && 'animate-spin')" />
          Refresh
        </Button>
        <Button @click="openCreate">
          <Plus />
          Add device
        </Button>
      </div>
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
            <TableHead class="w-12 px-4"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="loading">
            <TableRow v-for="i in 5" :key="i" class="hover:bg-transparent">
              <TableCell v-for="c in 6" :key="c" class="px-4 py-3">
                <Skeleton class="h-4 w-full max-w-32" />
              </TableCell>
            </TableRow>
          </template>

          <TableRow v-else-if="devices.length === 0" class="hover:bg-transparent">
            <TableCell colspan="6" class="px-4 py-16">
              <div class="flex flex-col items-center gap-2 text-center">
                <span class="bg-muted text-muted-foreground grid size-10 place-items-center rounded-full">
                  <MonitorSmartphone class="size-5" />
                </span>
                <p class="font-medium">No devices yet</p>
                <p class="text-muted-foreground text-sm">
                  Add a device to generate an API key for the agent.
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
              <TableCell class="px-4 py-3 text-right" @click.stop>
                <DropdownMenu>
                  <DropdownMenuTrigger as-child>
                    <Button variant="ghost" size="icon" :disabled="busy">
                      <MoreHorizontal />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" class="w-44">
                    <DropdownMenuItem @select="rotate(d)">
                      <KeyRound />
                      Rotate key
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem variant="destructive" @select="remove(d)">
                      <Trash2 />
                      Delete device
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          </template>
        </TableBody>
      </Table>
    </Card>

    <!-- Add device -->
    <div
      v-if="showCreate"
      class="fixed inset-0 z-50 grid place-items-center bg-black/50 p-4"
      @click.self="showCreate = false"
    >
      <Card class="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Add device</CardTitle>
          <CardDescription>
            Choose a stable id for the device, then copy its API key into the agent config.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form class="space-y-4" @submit.prevent="createDevice">
            <div class="space-y-2">
              <Label for="device-id">Device ID</Label>
              <Input
                id="device-id"
                v-model="newDeviceId"
                required
                maxlength="64"
                pattern="[a-zA-Z0-9][a-zA-Z0-9._-]*"
                placeholder="turn-01"
              />
            </div>
            <Alert v-if="createError" variant="destructive">
              <AlertCircle />
              <AlertDescription>{{ createError }}</AlertDescription>
            </Alert>
            <div class="flex justify-end gap-2">
              <Button type="button" variant="ghost" @click="showCreate = false">Cancel</Button>
              <Button type="submit" :disabled="busy || !newDeviceId.trim()">Create</Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>

    <!-- Issued API key -->
    <div
      v-if="issued"
      class="fixed inset-0 z-50 grid place-items-center bg-black/50 p-4"
      @click.self="closeIssued"
    >
      <Card class="w-full max-w-md">
        <CardHeader>
          <CardTitle>API key for {{ issued.device_id }}</CardTitle>
          <CardDescription>
            Copy this now — it is shown only once. Only a hash is stored on the server.
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="bg-muted flex items-center gap-2 rounded-md p-3">
            <code class="min-w-0 flex-1 overflow-x-auto text-xs whitespace-nowrap">{{
              issued.api_key
            }}</code>
            <Button variant="outline" size="icon" @click="copyKey">
              <Check v-if="copied" />
              <Copy v-else />
            </Button>
          </div>
          <div class="flex justify-end">
            <Button @click="closeIssued">
              <X />
              Done
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
