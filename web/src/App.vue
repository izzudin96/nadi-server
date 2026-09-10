<script setup lang="ts">
import { Activity, LogOut, User } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

async function logout() {
  await auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-svh bg-muted/30">
    <header
      v-if="auth.authenticated"
      class="bg-background/80 sticky top-0 z-40 border-b backdrop-blur"
    >
      <div class="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
        <router-link to="/" class="flex items-center gap-2 font-semibold">
          <span class="bg-primary text-primary-foreground grid size-7 place-items-center rounded-md">
            <Activity class="size-4" />
          </span>
          Nadi
        </router-link>

        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="icon" class="rounded-full">
              <Avatar>
                <AvatarFallback>
                  <User class="size-4" />
                </AvatarFallback>
              </Avatar>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-44">
            <DropdownMenuLabel>My account</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" @select="logout">
              <LogOut />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
    <router-view />
  </div>
</template>
