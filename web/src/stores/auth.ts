import { defineStore } from 'pinia'
import { api } from '../api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    authenticated: false,
  }),
  actions: {
    async register(email: string, password: string) {
      await api.register(email, password)
      this.authenticated = true
    },
    async login(email: string, password: string) {
      await api.login(email, password)
      this.authenticated = true
    },
    async logout() {
      try {
        await api.logout()
      } finally {
        this.authenticated = false
      }
    },
    async bootstrap() {
      try {
        await api.me()
        this.authenticated = true
      } catch {
        this.authenticated = false
      }
    },
  },
})
