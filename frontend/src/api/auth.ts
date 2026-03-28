import client from './client';
import type { ApiResponse, AuthTokens } from '@/types';

export async function register(username: string, email: string, password: string) {
  const { data } = await client.post<ApiResponse<AuthTokens>>('/auth/register', { username, email, password });
  return data.data;
}

export async function login(email: string, password: string) {
  const { data } = await client.post<ApiResponse<AuthTokens>>('/auth/login', { email, password });
  return data.data;
}

export async function logout() {
  await client.post('/auth/logout');
}

export async function revokeAllSessions() {
  await client.post('/auth/revoke-all');
}

export async function changePassword(currentPassword: string, newPassword: string) {
  await client.post('/auth/change-password', { current_password: currentPassword, new_password: newPassword });
}
