import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';
import type {
  CreateExportReq,
  CategoryStatResponse,
  DeptStatResponse,
  ExportJob,
  InventoryDiffReport,
} from '@/types/api';
import type { ApiResponse } from '@/types/common';
import { unwrapApiResponse } from '@/utils/api';

export const reportApi = createApi({
  reducerPath: 'reportApi',
  baseQuery: baseQueryWithReauth,
  tagTypes: ['Report'],
  endpoints: (builder) => ({
    getAssetsByDept: builder.query<DeptStatResponse, void>({
      query: () => '/report/assets/by-dept',
      transformResponse: (response: ApiResponse<DeptStatResponse>) => unwrapApiResponse(response),
      providesTags: [{ type: 'Report', id: 'BY_DEPT' }],
    }),
    getAssetsByCategory: builder.query<CategoryStatResponse, { departmentId?: number } | void>({
      query: (params) => ({
        url: '/report/assets/by-category',
        params: params ?? {},
      }),
      transformResponse: (response: ApiResponse<CategoryStatResponse>) => unwrapApiResponse(response),
      providesTags: [{ type: 'Report', id: 'BY_CATEGORY' }],
    }),
    getInventoryDiff: builder.query<InventoryDiffReport, number>({
      query: (taskId) => `/report/inventory/diff/${taskId}`,
      transformResponse: (response: ApiResponse<InventoryDiffReport>) => unwrapApiResponse(response),
      providesTags: (_r, _e, taskId) => [{ type: 'Report', id: `DIFF_${taskId}` }],
    }),
    createExport: builder.mutation<{ jobId: number }, CreateExportReq>({
      query: (body) => ({ url: '/report/export', method: 'POST', body }),
      transformResponse: (response: ApiResponse<{ jobId: number }>) => unwrapApiResponse(response),
    }),
    getExportStatus: builder.query<ExportJob, number>({
      query: (jobId) => `/report/export/${jobId}`,
      transformResponse: (response: ApiResponse<{ jobId: number; status: ExportJob['status']; errorMessage?: string | null }>) => {
        const data = unwrapApiResponse(response);
        return {
          jobId: data.jobId,
          status: data.status,
          errorMessage: data.errorMessage ?? null,
          downloadUrl: `/api/v1/report/export/${data.jobId}/download`,
        };
      },
    }),
  }),
});

export const {
  useGetAssetsByDeptQuery,
  useGetAssetsByCategoryQuery,
  useGetInventoryDiffQuery,
  useCreateExportMutation,
  useGetExportStatusQuery,
} = reportApi;
