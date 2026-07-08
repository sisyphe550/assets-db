import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithReauth } from './baseQuery';
import type {
  Asset,
  AssetListParams,
  CreateAssetReq,
  UpdateAssetReq,
} from '@/types/api';
import type { ApiResponse, PaginatedData } from '@/types/common';
import { unwrapApiResponse } from '@/utils/api';

export const assetApi = createApi({
  reducerPath: 'assetApi',
  baseQuery: baseQueryWithReauth,
  tagTypes: ['AssetList', 'SharedAssetList', 'Asset'],
  endpoints: (builder) => ({
    getAssets: builder.query<PaginatedData<Asset>, AssetListParams | void>({
      query: (params) => ({
        url: '/asset/assets',
        params: params ?? {},
      }),
      transformResponse: (response: ApiResponse<PaginatedData<Asset>>) =>
        unwrapApiResponse(response),
      providesTags: (result) =>
        result
          ? [
              ...result.list.map((item) => ({ type: 'Asset' as const, id: item.id })),
              { type: 'AssetList', id: 'LIST' },
            ]
          : [{ type: 'AssetList', id: 'LIST' }],
    }),
    getAsset: builder.query<Asset, number>({
      query: (id) => `/asset/assets/${id}`,
      transformResponse: (response: ApiResponse<Asset>) => unwrapApiResponse(response),
      providesTags: (_result, _err, id) => [{ type: 'Asset', id }],
    }),
    createAsset: builder.mutation<Asset, CreateAssetReq>({
      query: (body) => ({ url: '/asset/assets', method: 'POST', body }),
      transformResponse: (response: ApiResponse<Asset>) => unwrapApiResponse(response),
      invalidatesTags: [{ type: 'AssetList', id: 'LIST' }],
    }),
    updateAsset: builder.mutation<Asset, { id: number; body: UpdateAssetReq }>({
      query: ({ id, body }) => ({ url: `/asset/assets/${id}`, method: 'PUT', body }),
      transformResponse: (response: ApiResponse<Asset>) => unwrapApiResponse(response),
      invalidatesTags: (_result, _err, { id }) => [
        { type: 'AssetList', id: 'LIST' },
        { type: 'Asset', id },
      ],
    }),
    deleteAsset: builder.mutation<void, number>({
      query: (id) => ({ url: `/asset/assets/${id}`, method: 'DELETE' }),
      transformResponse: (response: ApiResponse<null>) => {
        unwrapApiResponse(response);
      },
      invalidatesTags: [{ type: 'AssetList', id: 'LIST' }],
    }),
    getSharedAssets: builder.query<PaginatedData<Asset>, AssetListParams | void>({
      query: (params) => ({
        url: '/asset/assets/shared',
        params: params ?? {},
      }),
      transformResponse: (response: ApiResponse<PaginatedData<Asset>>) =>
        unwrapApiResponse(response),
      providesTags: [{ type: 'SharedAssetList', id: 'LIST' }],
    }),
  }),
});

export const {
  useGetAssetsQuery,
  useLazyGetAssetsQuery,
  useGetAssetQuery,
  useCreateAssetMutation,
  useUpdateAssetMutation,
  useDeleteAssetMutation,
  useGetSharedAssetsQuery,
} = assetApi;
