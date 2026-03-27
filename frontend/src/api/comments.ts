import client from './client';
import type { ApiResponse, Comment, PaginationResponse } from '@/types';

export async function createComment(postId: string, content: string, parentId = '', mediaUrl = '') {
  const { data } = await client.post<ApiResponse<{ comment: Comment }>>(`/posts/${postId}/comments`, { content, parent_id: parentId, media_url: mediaUrl });
  return data.data.comment;
}

export async function getCommentTree(postId: string, cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ comments: Comment[]; pagination: PaginationResponse }>>(`/posts/${postId}/comments${params}`);
  return data.data;
}

export async function deleteComment(id: string) { await client.delete(`/comments/${id}`); }
export async function likeComment(id: string) { await client.post(`/comments/${id}/like`); }
export async function unlikeComment(id: string) { await client.delete(`/comments/${id}/like`); }
