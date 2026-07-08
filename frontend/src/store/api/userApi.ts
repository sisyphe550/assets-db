import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';
import type { DeptTreeNode, UserListItem, UserListParams } from '@/types/api';
import type { ApiResponse, PaginatedData } from '@/types/common';
import { unwrapApiResponse } from '@/utils/api';

export const userApi = createApi({
  reducerPath: 'userApi',
  baseQuery: baseQueryWithReauth,
  tagTypes: ['DeptTree', 'UserList'],
  endpoints: (builder) => ({
    getDeptTree: builder.query<DeptTreeNode[], void>({
      query: () => '/user/departments/tree',
      transformResponse: (response: ApiResponse<DeptTreeNode[]>) => unwrapApiResponse(response),
      providesTags: [{ type: 'DeptTree', id: 'TREE' }],
    }),
    listUsers: builder.query<PaginatedData<UserListItem>, UserListParams | void>({
      query: (params) => ({
        url: '/user/users',
        params: params ?? {},
      }),
      transformResponse: (response: ApiResponse<PaginatedData<UserListItem>>) =>
        unwrapApiResponse(response),
      providesTags: [{ type: 'UserList', id: 'LIST' }],
    }),
  }),
});

export const { useGetDeptTreeQuery, useListUsersQuery } = userApi;
