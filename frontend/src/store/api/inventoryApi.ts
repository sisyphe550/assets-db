import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';
import type {
  CreateInventoryTaskReq,
  ExpectedAsset,
  InventoryRecord,
  InventoryTask,
  SubmitItem,
  SubmitResult,
} from '@/types/api';
import type { ApiResponse, ListParams, PaginatedData } from '@/types/common';
import { unwrapApiResponse } from '@/utils/api';

export interface InventoryListParams extends ListParams {
  status?: number;
  scope?: 'assigned';
}

export const inventoryApi = createApi({
  reducerPath: 'inventoryApi',
  baseQuery: baseQueryWithReauth,
  tagTypes: ['InventoryList', 'Inventory', 'InventoryRecords'],
  endpoints: (builder) => ({
    getTasks: builder.query<PaginatedData<InventoryTask>, InventoryListParams | void>({
      query: (params) => ({
        url: '/inventory/tasks',
        params: params ?? {},
      }),
      transformResponse: (response: ApiResponse<PaginatedData<InventoryTask>>) =>
        unwrapApiResponse(response),
      providesTags: (result) =>
        result
          ? [
              ...result.list.map((item) => ({ type: 'Inventory' as const, id: item.id })),
              { type: 'InventoryList', id: 'LIST' },
            ]
          : [{ type: 'InventoryList', id: 'LIST' }],
    }),
    getTask: builder.query<InventoryTask, number>({
      query: (id) => `/inventory/tasks/${id}`,
      transformResponse: (response: ApiResponse<InventoryTask>) => unwrapApiResponse(response),
      providesTags: (_r, _e, id) => [{ type: 'Inventory', id }],
    }),
    createTask: builder.mutation<InventoryTask, CreateInventoryTaskReq>({
      query: (body) => ({ url: '/inventory/tasks', method: 'POST', body }),
      transformResponse: (response: ApiResponse<Record<string, unknown>>, _meta, arg) => {
        const data = unwrapApiResponse(response);
        return {
          id: Number(data.taskId ?? data.id),
          taskName: String(data.taskName),
          scopeDeptId: Number(data.scopeDeptId),
          creatorId: 0,
          startTime: arg.startTime,
          endTime: arg.endTime,
          status: 1 as const,
          assigneeIds: (data.assigneeIds as number[]) ?? arg.assigneeIds,
          expectedAssetCount: Number(data.expectedAssetCount ?? 0),
          submittedCount: 0,
        };
      },
      invalidatesTags: [{ type: 'InventoryList', id: 'LIST' }],
    }),
    getExpectedAssets: builder.query<{ list: ExpectedAsset[]; total: number }, number>({
      query: (id) => `/inventory/tasks/${id}/expected-assets`,
      transformResponse: (
        response: ApiResponse<{ list: ExpectedAsset[]; total: number }>,
      ) => unwrapApiResponse(response),
    }),
    submitRecords: builder.mutation<SubmitResult, { taskId: number; items: SubmitItem[] }>({
      query: ({ taskId, items }) => ({
        url: `/inventory/tasks/${taskId}/submit`,
        method: 'POST',
        body: { items },
      }),
      transformResponse: (response: ApiResponse<SubmitResult>) => unwrapApiResponse(response),
      invalidatesTags: (_r, _e, { taskId }) => [
        { type: 'Inventory', id: taskId },
        { type: 'InventoryList', id: 'LIST' },
      ],
    }),
    archiveTask: builder.mutation<
      { taskId: number; archivedRecordCount: number; comparisonJobQueued: boolean },
      { id: number; force?: boolean }
    >({
      query: ({ id, force = false }) => ({
        url: `/inventory/tasks/${id}/archive`,
        method: 'POST',
        body: { force },
      }),
      transformResponse: (
        response: ApiResponse<{
          taskId: number;
          archivedRecordCount: number;
          comparisonJobQueued: boolean;
        }>,
      ) => unwrapApiResponse(response),
      invalidatesTags: (_r, _e, { id }) => [
        { type: 'Inventory', id },
        { type: 'InventoryList', id: 'LIST' },
      ],
    }),
    getRecords: builder.query<
      PaginatedData<InventoryRecord>,
      { taskId: number; page?: number; pageSize?: number; diffStatus?: number }
    >({
      query: ({ taskId, ...params }) => ({
        url: `/inventory/tasks/${taskId}/records`,
        params,
      }),
      transformResponse: (response: ApiResponse<PaginatedData<InventoryRecord>>) =>
        unwrapApiResponse(response),
      providesTags: (_r, _e, { taskId }) => [{ type: 'InventoryRecords', id: taskId }],
    }),
  }),
});

export const {
  useGetTasksQuery,
  useGetTaskQuery,
  useCreateTaskMutation,
  useGetExpectedAssetsQuery,
  useSubmitRecordsMutation,
  useArchiveTaskMutation,
  useGetRecordsQuery,
} = inventoryApi;
