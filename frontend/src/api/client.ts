import axios from 'axios';
import { useAuthStore } from '@/store/auth';

const client = axios.create({ baseURL: '/api' });

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

client.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config;
    if (error.response?.status === 401 && !original._retry) {
      original._retry = true;
      const refreshToken = localStorage.getItem('refresh_token');
      if (!refreshToken) {
        useAuthStore.getState().logout();
        return Promise.reject(error);
      }
      try {
        const { data } = await axios.post('/api/auth/refresh', { refresh_token: refreshToken });
        const newAccess = data.data.access_token;
        const newRefresh = data.data.refresh_token;
        useAuthStore.getState().setTokens(newAccess, newRefresh);
        original.headers.Authorization = `Bearer ${newAccess}`;
        return client(original);
      } catch {
        useAuthStore.getState().logout();
      }
    }
    return Promise.reject(error);
  }
);

export default client;
