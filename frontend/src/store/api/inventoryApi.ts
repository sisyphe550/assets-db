import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';
import type {
  CreateInventoryTaskReq,
  ExpectedAsset,
  InventoryConflict,
  InventoryDraft,
  InventoryRecord,
  InventoryTask,
  ResolveConflictReq,
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
  tagTypes: ['InventoryList', 'Inventory', 'InventoryRecords', 'InventoryDrafts', 'InventoryConflicts'],
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
      keepUnusedDataFor: 120,
    }),
    getDrafts: builder.query<{ list: InventoryDraft[]; total: number }, number>({
      query: (id) => `/inventory/tasks/${id}/drafts`,
      transformResponse: (
        response: ApiResponse<{ list: InventoryDraft[]; total: number }>,
      ) => unwrapApiResponse(response),
      providesTags: (_r, _e, id) => [{ type: 'InventoryDrafts', id }],
      keepUnusedDataFor: 120,
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
        { type: 'InventoryDrafts', id: taskId },
      ],
    }),
    archiveTask: builder.mutation<
      {
        taskId: number;
        archivedRecordCount: number;
        comparisonJobQueued: boolean;
        conflictCount?: number;
        pendingConflictCount?: number;
      },
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
          conflictCount?: number;
          pendingConflictCount?: number;
        }>,
      ) => unwrapApiResponse(response),
      invalidatesTags: (_r, _e, { id }) => [
        { type: 'Inventory', id },
        { type: 'InventoryList', id: 'LIST' },
        { type: 'InventoryConflicts', id },
      ],
    }),
    compareTask: builder.mutation<
      { taskId: number; status: number; compared?: boolean; alreadyDone?: boolean },
      number
    >({
      query: (id) => ({
        url: `/inventory/tasks/${id}/compare`,
        method: 'POST',
      }),
      transformResponse: (
        response: ApiResponse<{
          taskId: number;
          status: number;
          compared?: boolean;
          alreadyDone?: boolean;
        }>,
      ) => unwrapApiResponse(response),
      invalidatesTags: (_r, _e, id) => [
        { type: 'Inventory', id },
        { type: 'InventoryList', id: 'LIST' },
        { type: 'InventoryRecords', id },
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
    getConflicts: builder.query<
      { list: InventoryConflict[]; total: number; pendingCount: number },
      number
    >({
      query: (taskId) => `/inventory/tasks/${taskId}/conflicts`,
      transformResponse: (
        response: ApiResponse<{ list: InventoryConflict[]; total: number; pendingCount: number }>,
      ) => unwrapApiResponse(response),
      providesTags: (_r, _e, taskId) => [{ type: 'InventoryConflicts', id: taskId }],
    }),
    resolveConflict: builder.mutation<
      {
        assetNo: string;
        pendingConflictCount: number;
        comparisonJobQueued: boolean;
        allResolved: boolean;
      },
      { taskId: number; assetNo: string; body: ResolveConflictReq }
    >({
      query: ({ taskId, assetNo, body }) => ({
        url: `/inventory/tasks/${taskId}/conflicts/${encodeURIComponent(assetNo)}/resolve`,
        method: 'POST',
        body,
      }),
      transformResponse: (
        response: ApiResponse<{
          assetNo: string;
          pendingConflictCount: number;
          comparisonJobQueued: boolean;
          allResolved: boolean;
        }>,
      ) => unwrapApiResponse(response),
      invalidatesTags: (_r, _e, { taskId }) => [
        { type: 'Inventory', id: taskId },
        { type: 'InventoryConflicts', id: taskId },
        { type: 'InventoryRecords', id: taskId },
        { type: 'InventoryList', id: 'LIST' },
      ],
    }),
  }),
});

export const {
  useGetTasksQuery,
  useGetTaskQuery,
  useCreateTaskMutation,
  useGetExpectedAssetsQuery,
  useGetDraftsQuery,
  useSubmitRecordsMutation,
  useArchiveTaskMutation,
  useCompareTaskMutation,
  useGetRecordsQuery,
  useGetConflictsQuery,
  useResolveConflictMutation,
} = inventoryApi;
