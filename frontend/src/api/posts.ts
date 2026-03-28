import client from './client';
import type { ApiResponse, PaginationResponse, Post, TrendingHashtag } from '@/types';

export async function createPost(content: string, mediaUrls: string[] = []) {
  const { data } = await client.post<ApiResponse<{ post: Post }>>('/posts', { content, media_urls: mediaUrls });
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

export async function getGlobalFeed(cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ posts: Post[]; pagination: PaginationResponse }>>(`/feed/global${params}`);
  return data.data;
}

export async function getUserPosts(userId: string, cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ posts: Post[]; pagination: PaginationResponse }>>(`/users/${userId}/posts${params}`);
  return data.data;
}

export async function likePost(id: string) { await client.post(`/posts/${id}/like`); }
export async function unlikePost(id: string) { await client.delete(`/posts/${id}/like`); }

export async function bookmarkPost(id: string) { await client.post(`/posts/${id}/bookmark`); }
export async function unbookmarkPost(id: string) { await client.delete(`/posts/${id}/bookmark`); }

export async function getBookmarks(cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ posts: Post[]; pagination: PaginationResponse }>>(`/bookmarks${params}`);
  return data.data;
}

export async function updatePost(id: string, content: string, mediaUrls: string[] = []) {
  const { data } = await client.put<ApiResponse<{ post: Post }>>(`/posts/${id}`, { content, media_urls: mediaUrls });
  return data.data.post;
}

export async function recordViews(postIds: string[]) {
  await client.post('/posts/views', { post_ids: postIds });
}

export async function getUserReposts(userId: string, cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ posts: Post[]; pagination: PaginationResponse }>>(`/users/${userId}/reposts${params}`);
  return data.data;
}

export async function getPostsByHashtag(tag: string, cursor = '') {
  const params = cursor ? `?cursor=${cursor}` : '';
  const { data } = await client.get<ApiResponse<{ posts: Post[]; pagination: PaginationResponse }>>(`/hashtags/${tag}/posts${params}`);
  return data.data;
}

export async function getTrendingHashtags() {
  const { data } = await client.get<ApiResponse<{ hashtags: TrendingHashtag[] }>>('/trending/hashtags');
  return data.data.hashtags || [];
}

export async function repostPost(id: string) { await client.post(`/posts/${id}/repost`); }
export async function unrepostPost(id: string) { await client.delete(`/posts/${id}/repost`); }

export async function quotePost(id: string, content: string) {
  const { data } = await client.post<ApiResponse<{ post: Post }>>(`/posts/${id}/quote`, { content });
  return data.data.post;
}
