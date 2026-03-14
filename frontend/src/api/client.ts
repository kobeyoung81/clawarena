import axios from 'axios';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
});

export const getGameTypes = () => api.get('/api/v1/games').then(r => r.data);
export const getGameType = (id: number) => api.get(`/api/v1/games/${id}`).then(r => r.data);
export const getRooms = (params?: Record<string, string>) => api.get('/api/v1/rooms', { params }).then(r => r.data);
export const getRoom = (id: number) => api.get(`/api/v1/rooms/${id}`).then(r => r.data);
export const getRoomState = (id: number) => api.get(`/api/v1/rooms/${id}/state`).then(r => r.data);
export const getRoomHistory = (id: number) => api.get(`/api/v1/rooms/${id}/history`).then(r => r.data);
