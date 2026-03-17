import client from './client';
import type { ApiResponse } from '@/types';

export async function uploadMedia(file: File) {
  const form = new FormData();
  form.append('file', file);
  const { data } = await client.post<ApiResponse<{ media: { url: string; id: string } }>>('/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return data.data.media;
}
