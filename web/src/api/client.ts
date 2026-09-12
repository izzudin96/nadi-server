export interface Device {
  device_id: string
  hostname: string
  os: string
  arch: string
  agent_version: string
  last_seen_at: string | null
  online: boolean
}

export interface LatestMetric {
  name: string
  value: number
  unit: string
  ts: string
}

export interface Point {
  ts: string
  value: number
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export interface DeviceKey {
  device_id: string
  api_key: string
}

export const api = {
  registrationStatus: () => request<{ open: boolean }>('/api/auth/registration'),
  register: (email: string, password: string) =>
    request<{ email: string }>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  login: (email: string, password: string) =>
    request<{ email: string }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  logout: () => request<void>('/api/auth/logout', { method: 'POST' }),
  me: () => request<{ user_id: number }>('/api/auth/me'),
  devices: () => request<{ devices: Device[] }>('/api/devices'),
  createDevice: (deviceId: string) =>
    request<DeviceKey>('/api/devices', {
      method: 'POST',
      body: JSON.stringify({ device_id: deviceId }),
    }),
  rotateDevice: (id: string) =>
    request<DeviceKey>(`/api/devices/${encodeURIComponent(id)}/rotate`, { method: 'POST' }),
  deleteDevice: (id: string) =>
    request<void>(`/api/devices/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  latest: (id: string) =>
    request<{ device_id: string; metrics: LatestMetric[] }>(`/api/devices/${id}/latest`),
  series: (id: string, name: string, from: string, to: string) =>
    request<{ points: Point[] }>(
      `/api/devices/${id}/metrics/${encodeURIComponent(name)}?from=${from}&to=${to}`,
    ),
}
