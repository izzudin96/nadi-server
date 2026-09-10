<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api/client'
import type { Device } from '../api/client'

const devices = ref<Device[]>([])
const error = ref('')
const loading = ref(true)

async function load() {
  loading.value = true
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
  if (!ts) return 'never'
  return new Date(ts).toLocaleString()
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-6">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-2xl font-semibold">Devices</h1>
      <button class="rounded bg-blue-600 px-3 py-2 text-sm text-white" @click="load">Refresh</button>
    </div>

    <p v-if="error" class="text-red-600">{{ error }}</p>
    <p v-if="loading" class="text-gray-500">Loading…</p>

    <div v-else-if="devices.length === 0" class="rounded bg-white p-8 text-center text-gray-500">
      No devices yet.
    </div>

    <div v-else class="overflow-hidden rounded-lg bg-white shadow">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Status</th>
            <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Device</th>
            <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Hostname</th>
            <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">OS</th>
            <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Last seen</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
          <tr
            v-for="d in devices"
            :key="d.device_id"
            class="cursor-pointer hover:bg-gray-50"
            @click="$router.push(`/devices/${d.device_id}`)"
          >
            <td class="px-4 py-3">
              <span
                class="inline-flex items-center gap-1.5 text-sm"
                :class="d.online ? 'text-green-600' : 'text-gray-400'"
              >
                <span
                  class="h-2 w-2 rounded-full"
                  :class="d.online ? 'bg-green-500' : 'bg-gray-300'"
                ></span>
                {{ d.online ? 'online' : 'offline' }}
              </span>
            </td>
            <td class="px-4 py-3 text-sm font-medium">{{ d.device_id }}</td>
            <td class="px-4 py-3 text-sm text-gray-600">{{ d.hostname || '—' }}</td>
            <td class="px-4 py-3 text-sm text-gray-600">{{ d.os || '—' }}</td>
            <td class="px-4 py-3 text-sm text-gray-600">{{ fmt(d.last_seen_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
