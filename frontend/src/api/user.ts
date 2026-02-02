/**
 * User API endpoints
 * Handles user profile management and password changes
 */

import { apiClient } from './client'
import type { User, ChangePasswordRequest } from '@/types'

/**
 * Get current user profile
 * @returns User profile data
 */
export async function getProfile(): Promise<User> {
  const { data } = await apiClient.get<User>('/user/profile')
  return data
}

/**
 * Update current user profile
 * @param profile - Profile data to update
 * @returns Updated user profile data
 */
export async function updateProfile(profile: {
  username?: string
}): Promise<User> {
  const { data } = await apiClient.put<User>('/user', profile)
  return data
}

/**
 * Change current user password
 * @param passwords - Old and new password
 * @returns Success message
 */
export async function changePassword(
  oldPassword: string,
  newPassword: string
): Promise<{ message: string }> {
  const payload: ChangePasswordRequest = {
    old_password: oldPassword,
    new_password: newPassword
  }

  const { data } = await apiClient.put<{ message: string }>('/user/password', payload)
  return data
}

// User attribute types
export interface UserAttributeDefinition {
  id: number
  key: string
  name: string
  description: string
  type: string
  options: { value: string; label: string }[]
  required: boolean
  placeholder: string
}

/**
 * Get user attribute definitions
 * @returns List of enabled attribute definitions
 */
export async function getAttributeDefinitions(): Promise<UserAttributeDefinition[]> {
  const { data } = await apiClient.get<UserAttributeDefinition[]>('/user/attributes/definitions')
  return data
}

/**
 * Get current user's attribute values
 * @returns Map of attribute ID to value
 */
export async function getMyAttributes(): Promise<Record<number, string>> {
  const { data } = await apiClient.get<Record<number, string>>('/user/attributes')
  return data
}

/**
 * Update current user's attribute values
 * @param attributes - Map of attribute ID to value
 * @returns Success message
 */
export async function updateMyAttributes(
  attributes: Record<number, string>
): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>('/user/attributes', { attributes })
  return data
}

/**
 * Get current user's invite count
 * @returns Invite count
 */
export async function getInviteCount(): Promise<{ invite_count: number }> {
  const { data } = await apiClient.get<{ invite_count: number }>('/user/invite-count')
  return data
}

/**
 * Agent attribute info
 */
export interface AgentAttribute {
  key: string
  name: string
  type?: string
  value: string
}

/**
 * Agent contact info (public fields only)
 */
export interface AgentContact {
  email: string
  username: string
  attributes?: AgentAttribute[]
}

/**
 * Agent contact response type
 */
export interface AgentContactResponse {
  has_agent: boolean
  agent?: AgentContact | null
}

/**
 * Get current user's agent contact info
 * @returns Agent contact info with attributes
 */
export async function getAgentContact(): Promise<AgentContactResponse> {
  const { data } = await apiClient.get<AgentContactResponse>('/user/agent-contact')
  return data
}

export const userAPI = {
  getProfile,
  updateProfile,
  changePassword,
  getAttributeDefinitions,
  getMyAttributes,
  updateMyAttributes,
  getInviteCount,
  getAgentContact
}

export default userAPI
