import { fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { BaseQueryFn, FetchArgs, FetchBaseQueryError } from '@reduxjs/toolkit/query';
import { message } from 'antd';
import type { RootState } from '@/store';
import { setCredentials, logout } from '@/store/slices/authSlice';
import type { TokenPair } from '@/types/api';
import type { ApiResponse } from '@/types/common';
import { unwrapApiResponse } from '@/utils/api';

const rawBaseQuery = fetchBaseQuery({
  baseUrl: '/api/v1',
  prepareHeaders: (headers, { getState }) => {
    const token = (getState() as RootState).auth.accessToken;
    if (token) {
      headers.set('Authorization', `Bearer ${token}`);
    }
    return headers;
  },
});

export const baseQueryWithReauth: BaseQueryFn<
  string | FetchArgs,
  unknown,
  FetchBaseQueryError
> = async (args, api, extraOptions) => {
  let result = await rawBaseQuery(args, api, extraOptions);

  if (result.error && result.error.status === 401) {
    const state = api.getState() as RootState;
    const refreshToken = state.auth.refreshToken;
    if (refreshToken) {
      const refreshResult = await rawBaseQuery(
        { url: '/user/refresh', method: 'POST', body: { refreshToken } },
        api,
        extraOptions,
      );
      if (refreshResult.data) {
        try {
          const data = unwrapApiResponse(refreshResult.data as ApiResponse<TokenPair>);
          api.dispatch(setCredentials(data));
          result = await rawBaseQuery(args, api, extraOptions);
          return result;
        } catch {
          // fall through to logout
        }
      }
    }
    api.dispatch(logout());
    message.error('登录已过期，请重新登录');
    window.location.href = '/login';
  }
  return result;
};
