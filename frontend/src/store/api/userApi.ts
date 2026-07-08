import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';
import type { DeptTreeNode } from '@/types/api';
import type { ApiResponse } from '@/types/common';
import { unwrapApiResponse } from '@/utils/api';

export const userApi = createApi({
  reducerPath: 'userApi',
  baseQuery: baseQueryWithReauth,
  tagTypes: ['DeptTree'],
  endpoints: (builder) => ({
    getDeptTree: builder.query<DeptTreeNode[], void>({
      query: () => '/user/departments/tree',
      transformResponse: (response: ApiResponse<DeptTreeNode[]>) => unwrapApiResponse(response),
      providesTags: [{ type: 'DeptTree', id: 'TREE' }],
    }),
  }),
});

export const { useGetDeptTreeQuery } = userApi;
