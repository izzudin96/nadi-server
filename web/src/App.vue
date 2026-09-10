<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()
const router = useRouter()

async function logout() {
  await auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <header v-if="auth.authenticated" class="bg-white shadow">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <router-link to="/" class="text-lg font-semibold">Nadi</router-link>
        <button class="text-sm text-gray-600 hover:text-gray-900" @click="logout">Log out</button>
      </div>
    </header>
    <router-view />
  </div>
</template>
