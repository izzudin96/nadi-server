<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()

const email = ref('')
const password = ref('')
const mode = ref<'login' | 'register'>('login')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    if (mode.value === 'login') {
      await auth.login(email.value, password.value)
    } else {
      await auth.register(email.value, password.value)
    }
    router.push('/')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Something went wrong'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center">
    <div class="w-full max-w-sm rounded-lg bg-white p-8 shadow">
      <h1 class="mb-6 text-2xl font-semibold">Nadi</h1>

      <div class="mb-6 flex gap-2">
        <button
          class="flex-1 rounded px-3 py-2 text-sm"
          :class="mode === 'login' ? 'bg-blue-600 text-white' : 'bg-gray-100'"
          @click="mode = 'login'"
        >
          Log in
        </button>
        <button
          class="flex-1 rounded px-3 py-2 text-sm"
          :class="mode === 'register' ? 'bg-blue-600 text-white' : 'bg-gray-100'"
          @click="mode = 'register'"
        >
          Register
        </button>
      </div>

      <form class="space-y-4" @submit.prevent="submit">
        <div>
          <label class="mb-1 block text-sm text-gray-700">Email</label>
          <input
            v-model="email"
            type="email"
            required
            class="w-full rounded border border-gray-300 px-3 py-2"
            placeholder="you@example.com"
          />
        </div>
        <div>
          <label class="mb-1 block text-sm text-gray-700">Password</label>
          <input
            v-model="password"
            type="password"
            required
            :minlength="mode === 'register' ? 8 : undefined"
            class="w-full rounded border border-gray-300 px-3 py-2"
            placeholder="********"
          />
        </div>

        <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

        <button
          type="submit"
          :disabled="loading"
          class="w-full rounded bg-blue-600 px-3 py-2 text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {{ loading ? 'Please wait…' : mode === 'login' ? 'Log in' : 'Create account' }}
        </button>
      </form>
    </div>
  </div>
</template>
