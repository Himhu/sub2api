/**
 * Admin Agents API endpoints
 * Handles agent management operations for administrators
 */

import { apiClient } from '../client'
import type { User, BasePaginationResponse } from '@/types'

/**
 * Agent invite statistics
 */
export interface InviteStats {
  agent_id: number
  total_invited: number
  invited_agents: number
  invited_users: number
  direct_invited: number
}

/**
 * Set agent status request
 */
export interface SetAgentStatusRequest {
  is_agent: boolean
  parent_agent_id?: number | null
}

/**
 * List all agents with pagination
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    search?: string
  }
): Promise<BasePaginationResponse<User>> {
  const { data } = await apiClient.get<BasePaginationResponse<User>>('/admin/agents', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

/**
 * Get agent by ID
 */
export async function getById(id: number): Promise<User> {
  const { data } = await apiClient.get<User>(`/admin/agents/${id}`)
  return data
}

/**
 * Set or unset agent status for a user
 */
export async function setAgentStatus(
  userId: number,
  request: SetAgentStatusRequest
): Promise<User> {
  const { data } = await apiClient.patch<User>(`/admin/agents/${userId}/status`, request)
  return data
}

/**
 * Get downline users for an agent
 */
export async function getDownline(
  agentId: number,
  page: number = 1,
  pageSize: number = 20
): Promise<BasePaginationResponse<User>> {
  const { data } = await apiClient.get<BasePaginationResponse<User>>(
    `/admin/agents/${agentId}/downline`,
    { params: { page, page_size: pageSize } }
  )
  return data
}

/**
 * Get invite statistics for an agent
 */
export async function getInviteStats(agentId: number): Promise<InviteStats> {
  const { data } = await apiClient.get<InviteStats>(`/admin/agents/${agentId}/invite-stats`)
  return data
}

const agentsAPI = {
  list,
  getById,
  setAgentStatus,
  getDownline,
  getInviteStats
}

export default agentsAPI
