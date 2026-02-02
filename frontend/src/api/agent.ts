/**
 * Agent Center API endpoints
 * Handles agent-specific operations for users who are agents
 */

import { apiClient } from './client'
import type { User, PaginatedResponse } from '@/types'

/**
 * Invite statistics for an agent
 */
export interface InviteStats {
  agent_id: number
  total_invited: number
  invited_agents: number
  invited_users: number
  direct_invited: number
}

/**
 * Get current agent's downline users
 * @param page - Page number
 * @param pageSize - Items per page
 * @returns Paginated list of downline users
 */
export async function getMyDownline(
  page = 1,
  pageSize = 10
): Promise<PaginatedResponse<User>> {
  const { data } = await apiClient.get<PaginatedResponse<User>>('/agent/downline', {
    params: { page, page_size: pageSize }
  })
  return data
}

/**
 * Get current agent's invite statistics
 * @returns Invite statistics
 */
export async function getMyInviteStats(): Promise<InviteStats> {
  const { data } = await apiClient.get<InviteStats>('/agent/stats')
  return data
}

export const agentAPI = {
  getMyDownline,
  getMyInviteStats
}

export default agentAPI
