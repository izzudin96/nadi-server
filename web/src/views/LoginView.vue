<script setup lang="ts">
import { AlertCircle, Activity, Loader2 } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const email = ref('')
const password = ref('')
const mode = ref('login')
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
  <div class="grid min-h-svh place-items-center px-4 py-10">
    <div class="w-full max-w-sm">
      <div class="mb-6 flex flex-col items-center gap-2 text-center">
        <span class="bg-primary text-primary-foreground grid size-10 place-items-center rounded-xl">
          <Activity class="size-5" />
        </span>
        <h1 class="text-xl font-semibold tracking-tight">Welcome to Nadi</h1>
        <p class="text-muted-foreground text-sm">Sign in to monitor your devices.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Account</CardTitle>
          <CardDescription>Use your email to continue.</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs v-model="mode" class="gap-4">
            <TabsList class="w-full">
              <TabsTrigger value="login">Log in</TabsTrigger>
              <TabsTrigger value="register">Register</TabsTrigger>
            </TabsList>

            <form class="space-y-4" @submit.prevent="submit">
              <div class="space-y-2">
                <Label for="email">Email</Label>
                <Input
                  id="email"
                  v-model="email"
                  type="email"
                  required
                  autocomplete="email"
                  placeholder="you@example.com"
                />
              </div>
              <div class="space-y-2">
                <Label for="password">Password</Label>
                <Input
                  id="password"
                  v-model="password"
                  type="password"
                  required
                  :minlength="mode === 'register' ? 8 : undefined"
                  :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
                  placeholder="••••••••"
                />
              </div>

              <Alert v-if="error" variant="destructive">
                <AlertCircle />
                <AlertDescription>{{ error }}</AlertDescription>
              </Alert>

              <Button type="submit" class="w-full" :disabled="loading">
                <Loader2 v-if="loading" class="animate-spin" />
                {{ loading ? 'Please wait…' : mode === 'login' ? 'Log in' : 'Create account' }}
              </Button>
            </form>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
