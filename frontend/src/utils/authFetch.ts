import type { TokenPair } from '@/types/api';
import type { ApiResponse } from '@/types/common';
import { storage } from '@/utils/storage';

let refreshPromise: Promise<TokenPair | null> | null = null;

export async function refreshAccessToken(refreshToken: string): Promise<TokenPair | null> {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      const res = await fetch('/api/v1/user/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken }),
      });
      if (!res.ok) return null;
      const json = (await res.json()) as ApiResponse<TokenPair>;
      if (json.code !== 0 || !json.data) return null;
      storage.setAccessToken(json.data.accessToken);
      storage.setRefreshToken(json.data.refreshToken);
      return json.data;
    } catch {
      return null;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

export async function fetchWithAuth(url: string, init: RequestInit = {}): Promise<Response> {
  const attempt = (token: string | null) =>
    fetch(url, {
      ...init,
      headers: {
        ...(init.headers as Record<string, string> | undefined),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    });

  let token = storage.getAccessToken();
  let res = await attempt(token);

  if (res.status !== 401) {
    return res;
  }

  const refreshToken = storage.getRefreshToken();
  if (!refreshToken) {
    return res;
  }

  const pair = await refreshAccessToken(refreshToken);
  if (!pair) {
    return res;
  }

  token = pair.accessToken;
  return attempt(token);
}
