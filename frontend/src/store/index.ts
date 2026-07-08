import { configureStore } from '@reduxjs/toolkit';
import authReducer from './slices/authSlice';
import uiReducer from './slices/uiSlice';
import { authApi } from './api/authApi';
import { assetApi } from './api/assetApi';
import { userApi } from './api/userApi';
import { workflowApi } from './api/workflowApi';

import { inventoryApi } from './api/inventoryApi';

export const store = configureStore({
  reducer: {
    auth: authReducer,
    ui: uiReducer,
    [authApi.reducerPath]: authApi.reducer,
    [assetApi.reducerPath]: assetApi.reducer,
    [userApi.reducerPath]: userApi.reducer,
    [workflowApi.reducerPath]: workflowApi.reducer,
    [inventoryApi.reducerPath]: inventoryApi.reducer,
  },
  middleware: (getDefault) =>
    getDefault().concat(
      authApi.middleware,
      assetApi.middleware,
      userApi.middleware,
      workflowApi.middleware,
      inventoryApi.middleware,
    ),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
