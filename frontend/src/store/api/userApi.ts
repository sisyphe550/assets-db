import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';
import type {
  CollegeSubtreeResponse,
  CreateDeptReq,
  CreateUserReq,
  DeptTreeNode,
  UpdateUserStatusReq,
  UserInfo,
  UserListItem,
  UserListParams,
} from '@/types/api';
import type { ApiResponse, PaginatedData } from '@/types/common';
import { unwrapApiResponse } from '@/utils/api';

export const userApi = createApi({
  reducerPath: 'userApi',
  baseQuery: baseQueryWithReauth,
  tagTypes: ['DeptTree', 'UserList', 'User'],
  endpoints: (builder) => ({
    getDeptTree: builder.query<DeptTreeNode[], void>({
      query: () => '/user/departments/tree',
      transformResponse: (response: ApiResponse<{ nodes: DeptTreeNode[] }>) =>
        unwrapApiResponse(response).nodes ?? [],
      providesTags: [{ type: 'DeptTree', id: 'TREE' }],
    }),
    createDept: builder.mutation<DeptTreeNode, CreateDeptReq>({
      query: (body) => ({ url: '/user/departments', method: 'POST', body }),
      transformResponse: (response: ApiResponse<DeptTreeNode>) => unwrapApiResponse(response),
      invalidatesTags: [{ type: 'DeptTree', id: 'TREE' }],
    }),
    getCollegeSubtree: builder.query<CollegeSubtreeResponse, void>({
      query: () => '/user/departments/college-subtree',
      transformResponse: (response: ApiResponse<CollegeSubtreeResponse>) =>
        unwrapApiResponse(response),
    }),
    listUsers: builder.query<PaginatedData<UserListItem>, UserListParams | void>({
      query: (params) => ({
        url: '/user/users',
        params: params ?? {},
      }),
      transformResponse: (response: ApiResponse<PaginatedData<UserListItem>>) =>
        unwrapApiResponse(response),
      providesTags: (result) =>
        result
          ? [
              ...result.list.map((item) => ({ type: 'User' as const, id: item.id })),
              { type: 'UserList', id: 'LIST' },
            ]
          : [{ type: 'UserList', id: 'LIST' }],
    }),
    getUser: builder.query<UserListItem, number>({
      query: (id) => `/user/users/${id}`,
      transformResponse: (response: ApiResponse<UserListItem>) => unwrapApiResponse(response),
      providesTags: (_r, _e, id) => [{ type: 'User', id }],
    }),
    createUser: builder.mutation<UserInfo, CreateUserReq>({
      query: (body) => ({ url: '/user/users', method: 'POST', body }),
      transformResponse: (response: ApiResponse<UserInfo>) => unwrapApiResponse(response),
      invalidatesTags: [{ type: 'UserList', id: 'LIST' }],
    }),
    updateUserStatus: builder.mutation<void, { id: number; body: UpdateUserStatusReq }>({
      query: ({ id, body }) => ({ url: `/user/users/${id}/status`, method: 'PUT', body }),
      transformResponse: (response: ApiResponse<null>) => {
        unwrapApiResponse(response);
      },
      invalidatesTags: (_r, _e, { id }) => [
        { type: 'UserList', id: 'LIST' },
        { type: 'User', id },
      ],
    }),
    forceLogout: builder.mutation<void, number>({
      query: (id) => ({ url: `/user/users/${id}/force-logout`, method: 'POST' }),
      transformResponse: (response: ApiResponse<null>) => {
        unwrapApiResponse(response);
      },
    }),
  }),
});

export const {
  useGetDeptTreeQuery,
  useCreateDeptMutation,
  useGetCollegeSubtreeQuery,
  useListUsersQuery,
  useGetUserQuery,
  useCreateUserMutation,
  useUpdateUserStatusMutation,
  useForceLogoutMutation,
} = userApi;
