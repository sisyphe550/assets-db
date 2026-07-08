import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Provider } from 'react-redux';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ConfigProvider, message } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { configureStore } from '@reduxjs/toolkit';
import LoginPage from '@/pages/login/LoginPage';
import authReducer from '@/store/slices/authSlice';
import uiReducer from '@/store/slices/uiSlice';
import { authApi } from '@/store/api/authApi';

const mockLoginUnwrap = vi.fn();
const mockGetMeUnwrap = vi.fn();

vi.mock('@/store/api/authApi', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/store/api/authApi')>();
  return {
    ...actual,
    useLoginMutation: () => [
      () => ({ unwrap: () => mockLoginUnwrap() }),
      { isLoading: false },
    ],
    useLazyGetMeQuery: () => [() => ({ unwrap: () => mockGetMeUnwrap() }), {}],
  };
});

function renderLogin(atPath = '/login') {
  const store = configureStore({
    reducer: {
      auth: authReducer,
      ui: uiReducer,
      [authApi.reducerPath]: authApi.reducer,
    },
    middleware: (getDefault) => getDefault().concat(authApi.middleware),
  });

  const view = render(
    <Provider store={store}>
      <ConfigProvider locale={zhCN}>
        <MemoryRouter initialEntries={[atPath]}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/admin/dashboard" element={<div>Admin Dashboard</div>} />
          </Routes>
        </MemoryRouter>
      </ConfigProvider>
    </Provider>,
  );

  return { ...view, store };
}

describe('LoginPage', () => {
  beforeEach(() => {
    message.destroy();
    mockLoginUnwrap.mockReset();
    mockGetMeUnwrap.mockReset();
  });

  it('renders login form', () => {
    renderLogin();
    expect(screen.getByText('高校固定资产管理系统')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('用户名/工号')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '登 录' })).toBeInTheDocument();
  });

  it('logs in and navigates to role home on success', async () => {
    const user = userEvent.setup();
    mockLoginUnwrap.mockResolvedValue({
      accessToken: 'test-access',
      refreshToken: 'test-refresh',
      expiresIn: 3600,
      tokenType: 'Bearer',
    });
    mockGetMeUnwrap.mockResolvedValue({
      id: 1,
      username: 'admin_school',
      realName: '校级管理员',
      roleLevel: 1,
      departmentId: 1,
      departmentName: '学校',
      status: 1,
    });

    const { store } = renderLogin();
    await user.type(screen.getByPlaceholderText('用户名/工号'), 'admin_school');
    await user.type(screen.getByPlaceholderText('密码'), 'Test@123456');
    await user.click(screen.getByRole('button', { name: '登 录' }));

    await waitFor(() => {
      expect(store.getState().auth.isAuthenticated).toBe(true);
      expect(screen.getByText('Admin Dashboard')).toBeInTheDocument();
    });
  });

  it('shows error message on invalid credentials', async () => {
    const user = userEvent.setup();
    const errorSpy = vi.spyOn(message, 'error');
    mockLoginUnwrap.mockImplementation(() =>
      Promise.reject({ code: 40101, message: '用户名或密码错误' }),
    );

    renderLogin();
    await user.type(screen.getByPlaceholderText('用户名/工号'), 'wrong_user');
    await user.type(screen.getByPlaceholderText('密码'), 'wrong_pass');
    await user.click(screen.getByRole('button', { name: '登 录' }));

    await waitFor(() => {
      expect(errorSpy).toHaveBeenCalledWith('用户名或密码错误');
    });
    expect(mockGetMeUnwrap).not.toHaveBeenCalled();
    errorSpy.mockRestore();
  });
});
