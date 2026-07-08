import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';
import { assetApi } from './assetApi';
import type {
  ApproveWorkflowReq,
  CreateWorkflowReq,
  WorkflowDetail,
  WorkflowListParams,
  WorkflowRequest,
} from '@/types/api';
import type { ApiResponse, PaginatedData } from '@/types/common';
import { unwrapApiResponse } from '@/utils/api';

export const workflowApi = createApi({
  reducerPath: 'workflowApi',
  baseQuery: baseQueryWithReauth,
  tagTypes: ['WorkflowList', 'Workflow'],
  endpoints: (builder) => ({
    getRequests: builder.query<PaginatedData<WorkflowRequest>, WorkflowListParams>({
      query: (params) => ({
        url: '/workflow/requests',
        params,
      }),
      transformResponse: (response: ApiResponse<PaginatedData<WorkflowRequest>>) =>
        unwrapApiResponse(response),
      providesTags: (result) =>
        result
          ? [
              ...result.list.map((item) => ({ type: 'Workflow' as const, id: item.id })),
              { type: 'WorkflowList', id: 'LIST' },
            ]
          : [{ type: 'WorkflowList', id: 'LIST' }],
    }),
    getRequest: builder.query<WorkflowDetail, number>({
      query: (id) => `/workflow/requests/${id}`,
      transformResponse: (response: ApiResponse<WorkflowDetail>) => unwrapApiResponse(response),
      providesTags: (_result, _err, id) => [{ type: 'Workflow', id }],
    }),
    createRequest: builder.mutation<WorkflowRequest, CreateWorkflowReq>({
      query: (body) => ({ url: '/workflow/requests', method: 'POST', body }),
      transformResponse: (response: ApiResponse<WorkflowRequest>) => unwrapApiResponse(response),
      invalidatesTags: [{ type: 'WorkflowList', id: 'LIST' }],
    }),
    approveRequest: builder.mutation<void, { id: number; body?: ApproveWorkflowReq }>({
      query: ({ id, body }) => ({
        url: `/workflow/requests/${id}/approve`,
        method: 'POST',
        body: body ?? {},
      }),
      transformResponse: (response: ApiResponse<null>) => {
        unwrapApiResponse(response);
      },
      invalidatesTags: (_r, _e, { id }) => [
        { type: 'WorkflowList', id: 'LIST' },
        { type: 'Workflow', id },
      ],
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        await queryFulfilled;
        dispatch(assetApi.util.invalidateTags([{ type: 'AssetList', id: 'LIST' }]));
      },
    }),
    rejectRequest: builder.mutation<void, { id: number; body?: ApproveWorkflowReq }>({
      query: ({ id, body }) => ({
        url: `/workflow/requests/${id}/reject`,
        method: 'POST',
        body: body ?? {},
      }),
      transformResponse: (response: ApiResponse<null>) => {
        unwrapApiResponse(response);
      },
      invalidatesTags: (_r, _e, { id }) => [
        { type: 'WorkflowList', id: 'LIST' },
        { type: 'Workflow', id },
      ],
    }),
  }),
});

export const {
  useGetRequestsQuery,
  useGetRequestQuery,
  useCreateRequestMutation,
  useApproveRequestMutation,
  useRejectRequestMutation,
} = workflowApi;
