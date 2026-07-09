import { describe, it, expect, beforeEach } from 'vitest';
import authReducer, { setCredentials, setUser, logout } from '@/store/slices/authSlice';
import type { UserInfo } from '@/types/api';

const mockUser: UserInfo = {
  id: 1,
  username: 'admin_school',
  realName: '校级管理员',
  roleLevel: 1,
  departmentId: 1,
  departmentName: '学校',
  status: 1,
};

describe('authSlice', () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it('sets credentials and persists tokens', () => {
    const state = authReducer(
      undefined,
      setCredentials({
        accessToken: 'access',
        refreshToken: 'refresh',
        expiresIn: 3600,
        tokenType: 'Bearer',
      }),
    );
    expect(state.isAuthenticated).toBe(true);
    expect(state.accessToken).toBe('access');
    expect(sessionStorage.getItem('fams_access_token')).toBe('access');
  });

  it('sets user profile', () => {
    const state = authReducer(undefined, setUser(mockUser));
    expect(state.user).toEqual(mockUser);
  });

  it('logout clears state and storage', () => {
    let state = authReducer(
      undefined,
      setCredentials({
        accessToken: 'access',
        refreshToken: 'refresh',
        expiresIn: 3600,
        tokenType: 'Bearer',
      }),
    );
    state = authReducer(state, setUser(mockUser));
    state = authReducer(state, logout());
    expect(state.isAuthenticated).toBe(false);
    expect(state.user).toBeNull();
    expect(sessionStorage.getItem('fams_access_token')).toBeNull();
  });
});
