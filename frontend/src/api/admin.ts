import client from './client';
import type { ApiResponse, User, PaginationResponse, LoginHistoryEntry } from '@/types';

interface ListUsersData {
  users: User[];
  pagination: PaginationResponse;
}

export async function listUsers(cursor?: string) {
  const params = cursor ? { cursor } : {};
  const { data } = await client.get<ApiResponse<ListUsersData>>('/admin/users', { params });
  return data.data;
}

export async function setAdmin(userId: string, isAdmin: boolean) {
  const { data } = await client.post<ApiResponse<User>>(`/admin/users/${userId}/set-admin`, { is_admin: isAdmin });
  return data.data;
}

export async function banUser(userId: string, isBanned: boolean) {
  const { data } = await client.post<ApiResponse<User>>(`/admin/users/${userId}/ban`, { is_banned: isBanned });
  return data.data;
}

export async function adminDeletePost(postId: string) {
  await client.delete(`/admin/posts/${postId}`);
}

export async function adminDeleteComment(commentId: string) {
  await client.delete(`/admin/comments/${commentId}`);
}

export async function getLoginHistory(userId: string): Promise<LoginHistoryEntry[]> {
  const { data } = await client.get<ApiResponse<LoginHistoryEntry[]>>(`/admin/users/${userId}/login-history`);
  return data.data;
}
