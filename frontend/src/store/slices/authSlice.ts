import { createSlice, type PayloadAction } from '@reduxjs/toolkit';
import type { TokenPair, UserInfo } from '@/types/api';
import { storage } from '@/utils/storage';
import type { RootState } from '@/store';

export interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: UserInfo | null;
  isAuthenticated: boolean;
}

const initialState: AuthState = {
  accessToken: storage.getAccessToken(),
  refreshToken: storage.getRefreshToken(),
  user: null,
  isAuthenticated: !!storage.getAccessToken(),
};

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    setCredentials: (state, action: PayloadAction<TokenPair>) => {
      state.accessToken = action.payload.accessToken;
      state.refreshToken = action.payload.refreshToken;
      state.isAuthenticated = true;
      storage.setAccessToken(action.payload.accessToken);
      storage.setRefreshToken(action.payload.refreshToken);
    },
    setUser: (state, action: PayloadAction<UserInfo>) => {
      state.user = action.payload;
    },
    logout: (state) => {
      state.accessToken = null;
      state.refreshToken = null;
      state.user = null;
      state.isAuthenticated = false;
      storage.clear();
    },
  },
});

export const { setCredentials, setUser, logout } = authSlice.actions;
export const selectCurrentUser = (state: RootState) => state.auth.user;
export const selectIsAuthenticated = (state: RootState) => state.auth.isAuthenticated;
export const selectAccessToken = (state: RootState) => state.auth.accessToken;
export default authSlice.reducer;
