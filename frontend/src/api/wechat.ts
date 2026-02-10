import { apiClient } from './client'

export interface WeChatQRCodeResponse {
  scene_id: string
  qrcode_url: string
  expire_seconds: number
}

export interface WeChatShortCodeResponse {
  scene_id: string
  short_code: string
}

export interface WeChatScanStatusResponse {
  status: 'pending' | 'code_sent' | 'expired'
}

export async function createQRCode(): Promise<WeChatQRCodeResponse> {
  const { data } = await apiClient.post<WeChatQRCodeResponse>('/auth/wechat/qrcode')
  return data
}

export async function createShortCode(): Promise<WeChatShortCodeResponse> {
  const { data } = await apiClient.post<WeChatShortCodeResponse>('/auth/wechat/shortcode')
  return data
}

export async function checkScanStatus(sceneId: string): Promise<WeChatScanStatusResponse> {
  const { data } = await apiClient.get<WeChatScanStatusResponse>('/auth/wechat/scan-status', {
    params: { scene_id: sceneId }
  })
  return data
}

// User profile binding APIs

export interface WeChatBindStatus {
  bound: boolean
  openid_masked?: string
}

export interface WeChatBindResponse {
  scene_id: string
  short_code: string
}

export async function getBindStatus(): Promise<WeChatBindStatus> {
  const { data } = await apiClient.get<WeChatBindStatus>('/user/wechat/status')
  return data
}

export async function bindWeChat(password: string): Promise<WeChatBindResponse> {
  const { data } = await apiClient.post<WeChatBindResponse>('/user/wechat/bind', { password })
  return data
}

export async function confirmBindWeChat(sceneId: string, code: string): Promise<void> {
  await apiClient.post('/user/wechat/confirm', { scene_id: sceneId, code })
}

export async function unbindWeChat(password: string): Promise<void> {
  await apiClient.post('/user/wechat/unbind', { password })
}

export const wechatAPI = {
  createQRCode,
  createShortCode,
  checkScanStatus,
  getBindStatus,
  bindWeChat,
  confirmBindWeChat,
  unbindWeChat
}

export default wechatAPI
