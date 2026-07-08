import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';
import type { LoginRequest, TokenPair, UserInfo } from '@/types/api';
import type { ApiResponse } from '@/types/common';
import { unwrapApiResponse } from '@/utils/api';

export const authApi = createApi({
  reducerPath: 'authApi',
  baseQuery: baseQueryWithReauth,
  tagTypes: ['Me'],
  endpoints: (builder) => ({
    login: builder.mutation<TokenPair, LoginRequest>({
      query: (body) => ({ url: '/user/login', method: 'POST', body }),
      transformResponse: (response: ApiResponse<TokenPair>) => unwrapApiResponse(response),
    }),
    logout: builder.mutation<void, void>({
      query: () => ({ url: '/user/logout', method: 'POST' }),
      transformResponse: (response: ApiResponse<null>) => {
        unwrapApiResponse(response);
      },
    }),
    getMe: builder.query<UserInfo, void>({
      query: () => '/user/me',
      transformResponse: (response: ApiResponse<UserInfo>) => unwrapApiResponse(response),
      providesTags: ['Me'],
    }),
  }),
});

export const { useLoginMutation, useLogoutMutation, useGetMeQuery, useLazyGetMeQuery } = authApi;
