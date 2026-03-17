import client from './client';
import type { ApiResponse, PaginationResponse, Post } from '@/types';

export async function createPost(content: string, mediaUrl = '') {
  const { data } = await client.post<ApiResponse<{ post: Post }>>('/posts', { content, media_url: mediaUrl });
  return data.data.post;
}

export async function getPost(id: string) {
  const { data } = await client.get<ApiResponse<{ post: Post }>>(`/posts/${id}`);
  return data.data.post;
}

export async function deletePost(id: string) { await client.delete(`/posts/${id}`); }

export async function getFeed(cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ posts: Post[]; pagination: PaginationResponse }>>(`/feed${params}`);
  return data.data;
}

export async function getUserPosts(userId: string, cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ posts: Post[]; pagination: PaginationResponse }>>(`/users/${userId}/posts${params}`);
  return data.data;
}

export async function likePost(id: string) { await client.post(`/posts/${id}/like`); }
export async function unlikePost(id: string) { await client.delete(`/posts/${id}/like`); }
