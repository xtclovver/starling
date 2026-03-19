import client from './client';
import type { ApiResponse, PaginationResponse, User } from '@/types';

export async function getUser(id: string) {
  const { data } = await client.get<ApiResponse<{ user: User }>>(`/users/${id}`);
  return data.data.user;
}

export async function updateUser(id: string, fields: { display_name?: string; bio?: string; avatar_url?: string }) {
  const { data } = await client.put<ApiResponse<{ user: User }>>(`/users/${id}`, fields);
  return data.data.user;
}

export async function searchUsers(query: string, cursor = '') {
  const params = new URLSearchParams({ q: query });
  if (cursor) params.set('cursor', cursor);
  const { data } = await client.get<ApiResponse<{ users: User[]; pagination: PaginationResponse }>>(`/users/search?${params}`);
  return data.data;
}

export async function follow(userId: string) { await client.post(`/users/${userId}/follow`); }
export async function unfollow(userId: string) { await client.delete(`/users/${userId}/follow`); }

export async function getFollowers(userId: string, cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ users: User[]; pagination: PaginationResponse }>>(`/users/${userId}/followers${params}`);
  return data.data;
}

export async function getFollowing(userId: string, cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ users: User[]; pagination: PaginationResponse }>>(`/users/${userId}/following${params}`);
  return data.data;
}

export async function getRecommendedUsers() {
  const { data } = await client.get<ApiResponse<{ users: User[] }>>('/users/recommended');
  return data.data.users || [];
}
