import axios from 'axios';
import { getPortalBase } from '../config';
import type { GameListItem } from '../types';

export const api = axios.create({
  baseURL: '',
  withCredentials: true,
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    // getPortalBase() is called here (not at module level) so it reads the
    // runtime config loaded by loadConfig() before the app renders.
    const portalBase = getPortalBase();
    const authPageUrl = portalBase ? `${portalBase}/auth.html` : '';
    if (err.response?.status === 401 && authPageUrl) {
      const redirect = encodeURIComponent(window.location.href);
      window.location.href = `${authPageUrl}?redirect=${redirect}`;
    }
    return Promise.reject(err);
  }
);

export const getGameTypes = () => api.get('/api/v1/games').then(r => r.data);
export const getGameType = (id: number) => api.get(`/api/v1/games/${id}`).then(r => r.data);
export const getRooms = (params?: Record<string, string>) => api.get('/api/v1/rooms', { params }).then(r => r.data);
export const getRoom = (id: number) => api.get(`/api/v1/rooms/${id}`).then(r => r.data);
export const getRoomHistory = (id: number) => api.get(`/api/v1/rooms/${id}/history`).then(r => r.data);
export const getGameHistory = (id: number) => api.get(`/api/v1/games/${id}/history`).then(r => r.data);

export async function getGamesHistory(params?: {
  game_type_id?: number;
  status?: string;
  page?: number;
  per_page?: number;
}): Promise<{ games: GameListItem[]; total_count: number; page: number; per_page: number }> {
  const query: Record<string, string> = {};
  if (params?.game_type_id) query.game_type_id = String(params.game_type_id);
  if (params?.status) query.status = params.status;
  if (params?.page) query.page = String(params.page);
  if (params?.per_page) query.per_page = String(params.per_page);
  return api.get('/api/v1/games/history', { params: query }).then(r => r.data);
}
