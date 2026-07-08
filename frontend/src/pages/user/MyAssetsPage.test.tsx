import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { MemoryRouter } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import MyAssetsPage from '@/pages/user/MyAssetsPage';
import authReducer from '@/store/slices/authSlice';
import uiReducer from '@/store/slices/uiSlice';
import { authApi } from '@/store/api/authApi';
import { assetApi } from '@/store/api/assetApi';
import { userApi } from '@/store/api/userApi';

const mockMyUnwrap = vi.fn();
const mockSharedUnwrap = vi.fn();

vi.mock('@/store/api/assetApi', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/store/api/assetApi')>();
  return {
    ...actual,
    useGetAssetsQuery: () => ({
      data: mockMyUnwrap(),
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetSharedAssetsQuery: () => ({
      data: mockSharedUnwrap(),
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    }),
  };
});

function renderPage() {
  const store = configureStore({
    reducer: {
      auth: authReducer,
      ui: uiReducer,
      [authApi.reducerPath]: authApi.reducer,
      [assetApi.reducerPath]: assetApi.reducer,
      [userApi.reducerPath]: userApi.reducer,
    },
    middleware: (getDefault) =>
      getDefault().concat(authApi.middleware, assetApi.middleware, userApi.middleware),
  });

  return render(
    <Provider store={store}>
      <ConfigProvider locale={zhCN}>
        <MemoryRouter>
          <MyAssetsPage />
        </MemoryRouter>
      </ConfigProvider>
    </Provider>,
  );
}

describe('MyAssetsPage', () => {
  beforeEach(() => {
    mockMyUnwrap.mockReturnValue({
      list: [
        {
          id: 1,
          assetNo: 'EQUIP-2026-0001',
          name: '激光切割机',
          category: '设备',
          price: 150000,
          purchaseTime: '2025-09-01T00:00:00+08:00',
          location: '101',
          departmentId: 15,
          userId: 3,
          isShared: 0,
          status: 2,
        },
      ],
      page: 1,
      pageSize: 20,
      total: 1,
    });
    mockSharedUnwrap.mockReturnValue({ list: [], page: 1, pageSize: 20, total: 0 });
  });

  it('renders borrowed assets tab', () => {
    renderPage();
    expect(screen.getByText('我的资产')).toBeInTheDocument();
    expect(screen.getByText('激光切割机')).toBeInTheDocument();
    expect(screen.getByText('申请归还')).toBeInTheDocument();
  });
});
