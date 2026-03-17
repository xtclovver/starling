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
