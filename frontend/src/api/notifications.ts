import client from './client';
import type { ApiResponse, PaginationResponse, Notification } from '@/types';

export async function getNotifications(cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ notifications: Notification[]; pagination: PaginationResponse }>>(`/notifications${params}`);
  return data.data;
}

export async function getUnreadCount() {
  const { data } = await client.get<ApiResponse<{ count: number }>>('/notifications/unread');
  return data.data.count;
}

export async function markRead(id: string) { await client.post(`/notifications/${id}/read`); }
export async function markAllRead() { await client.post('/notifications/read-all'); }
