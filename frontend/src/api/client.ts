import axios from 'axios';

const AUTH_PAGE_URL = import.meta.env.VITE_AUTH_PAGE_URL || '';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  withCredentials: true,
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401 && AUTH_PAGE_URL) {
      const redirect = encodeURIComponent(window.location.href);
      window.location.href = `${AUTH_PAGE_URL}?redirect=${redirect}`;
    }
    return Promise.reject(err);
  }
);

export const getGameTypes = () => api.get('/api/v1/games').then(r => r.data);
export const getGameType = (id: number) => api.get(`/api/v1/games/${id}`).then(r => r.data);
export const getRooms = (params?: Record<string, string>) => api.get('/api/v1/rooms', { params }).then(r => r.data);
export const getRoom = (id: number) => api.get(`/api/v1/rooms/${id}`).then(r => r.data);
export const getRoomState = (id: number) => api.get(`/api/v1/rooms/${id}/state`).then(r => r.data);
export const getRoomHistory = (id: number) => api.get(`/api/v1/rooms/${id}/history`).then(r => r.data);
