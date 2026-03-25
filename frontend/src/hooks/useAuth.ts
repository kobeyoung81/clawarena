import { useQuery, useQueryClient } from '@tanstack/react-query';

const AUTH_BASE = import.meta.env.VITE_AUTH_BASE_URL || 'https://auth.losclaws.com';

interface HumanUser {
  id: string;
  name: string;
  email: string;
  email_verified: boolean;
  created_at: string;
}

async function fetchMe(): Promise<HumanUser | null> {
  let res = await fetch(`${AUTH_BASE}/auth/v1/humans/me`, {
    credentials: 'include',
  });
  if (res.status === 401) {
    // Access token expired — attempt silent refresh
    const refreshRes = await fetch(`${AUTH_BASE}/auth/v1/token/refresh`, {
      method: 'POST',
      credentials: 'include',
    });
    if (refreshRes.ok) {
      res = await fetch(`${AUTH_BASE}/auth/v1/humans/me`, {
        credentials: 'include',
      });
    }
  }
  if (!res.ok) return null;
  const data = await res.json();
  return data as HumanUser;
}

export function useAuth() {
  const queryClient = useQueryClient();

  const { data: user, isLoading } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: fetchMe,
    retry: 1,
    staleTime: 5 * 60 * 1000,
  });

  async function logout() {
    await fetch(`${AUTH_BASE}/auth/v1/humans/logout`, {
      method: 'POST',
      credentials: 'include',
    }).catch(() => {});
    queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
  }

  return { user: user ?? null, isLoading, logout };
}
