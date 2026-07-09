import { describe, it, expect, beforeEach } from 'vitest';
import { storage } from '@/utils/storage';

describe('storage', () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it('stores and retrieves access token in sessionStorage', () => {
    storage.setAccessToken('token-a');
    expect(storage.getAccessToken()).toBe('token-a');
    expect(sessionStorage.getItem('fams_access_token')).toBe('token-a');
  });

  it('stores and retrieves refresh token in sessionStorage', () => {
    storage.setRefreshToken('token-r');
    expect(storage.getRefreshToken()).toBe('token-r');
  });

  it('clears all tokens', () => {
    storage.setAccessToken('a');
    storage.setRefreshToken('r');
    storage.clear();
    expect(storage.getAccessToken()).toBeNull();
    expect(storage.getRefreshToken()).toBeNull();
  });
});
