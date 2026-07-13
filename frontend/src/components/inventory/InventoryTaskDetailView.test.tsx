import { ConfigProvider } from 'antd';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import zhCN from 'antd/locale/zh_CN';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import InventoryTaskDetailView from './InventoryTaskDetailView';

const mocks = vi.hoisted(() => ({
  compareTask: vi.fn(),
  currentUser: { id: 10001, roleLevel: 1 },
  currentTask: {
    id: 11,
    taskName: '盘点测试IA-3',
    scopeDeptId: 15,
    creatorId: 10001,
    startTime: '2026-07-13T00:00:00+08:00',
    endTime: '2026-07-20T23:59:00+08:00',
    status: 2,
    assigneeIds: [10003, 10004],
    expectedAssetCount: 4,
    submittedCount: 4,
    pendingConflictCount: 0,
  },
}));

vi.mock('@/store/hooks', () => ({
  useAppSelector: () => mocks.currentUser,
}));

vi.mock('@/components/inventory/InventoryConflictPanel', () => ({
  default: ({ taskId }: { taskId: number }) => <div>裁决面板-{taskId}</div>,
}));

vi.mock('@/components/inventory/InventorySpreadsheet', () => ({
  default: () => <div>盘点表格</div>,
}));

vi.mock('@/store/api/inventoryApi', () => ({
  useGetTaskQuery: () => ({
    data: mocks.currentTask,
    isLoading: false,
    isError: false,
    error: undefined,
    refetch: vi.fn(),
  }),
  useGetConflictsQuery: () => ({
    data: {
      list: [
        {
          assetNo: 'EQUIP-2026-0001',
          candidates: [
            { operatorId: 10003, actualLocation: '一号实验楼103' },
            { operatorId: 10004, actualLocation: '一号实验楼102' },
          ],
        },
      ],
      total: 1,
      pendingCount: 1,
    },
    isLoading: false,
  }),
  useGetExpectedAssetsQuery: () => ({
    data: { list: [{ assetId: 1, assetNo: 'EQUIP-1', name: '设备', bookLocation: 'A101' }], total: 1 },
    isLoading: false,
  }),
  useGetTaskItemsQuery: () => ({ data: undefined, isLoading: false, refetch: vi.fn() }),
  useUpdateTaskItemsMutation: () => [vi.fn(), { isLoading: false }],
  usePublishTaskMutation: () => [vi.fn(), { isLoading: false }],
  useGetDraftsQuery: () => ({ data: undefined, isLoading: false, refetch: vi.fn() }),
  useGetRecordsQuery: () => ({ data: undefined }),
  useSubmitRecordsMutation: () => [vi.fn(), { isLoading: false }],
  useArchiveTaskMutation: () => [vi.fn(), { isLoading: false }],
  useCompareTaskMutation: () => [mocks.compareTask, { isLoading: false }],
}));

function renderDetail(basePath: '/admin' | '/college' | '/user' = '/admin', showArchive = true) {
  return render(
    <ConfigProvider locale={zhCN}>
      <MemoryRouter initialEntries={[`${basePath}/inventory/tasks/11`]}>
        <Routes>
          <Route
            path={`${basePath}/inventory/tasks/:id`}
            element={<InventoryTaskDetailView basePath={basePath} showArchive={showArchive} />}
          />
        </Routes>
      </MemoryRouter>
    </ConfigProvider>,
  );
}

describe('InventoryTaskDetailView conflict arbitration', () => {
  beforeEach(() => {
    mocks.compareTask.mockReset();
    mocks.compareTask.mockReturnValue({ unwrap: () => Promise.resolve() });
    mocks.currentUser = { id: 10001, roleLevel: 1 };
    mocks.currentTask = {
      id: 11,
      taskName: '盘点测试IA-3',
      scopeDeptId: 15,
      creatorId: 10001,
      startTime: '2026-07-13T00:00:00+08:00',
      endTime: '2026-07-20T23:59:00+08:00',
      status: 2,
      assigneeIds: [10003, 10004],
      expectedAssetCount: 4,
      submittedCount: 4,
      // 模拟任务详情缓存/旧接口尚未带回最新冲突计数。
      pendingConflictCount: 0,
    };
  });

  it('uses the conflict endpoint as source of truth when task count is stale', () => {
    renderDetail();

    expect(screen.getByText('裁决面板-11')).toBeInTheDocument();
    expect(mocks.compareTask).not.toHaveBeenCalled();
  });

  it('keeps save draft visible for assigned users while hiding surplus rows', async () => {
    mocks.currentUser = { id: 10003, roleLevel: 3 };
    mocks.currentTask = {
      ...mocks.currentTask,
      status: 1,
      assigneeIds: [10003],
      submittedCount: 0,
      pendingConflictCount: 0,
    };

    renderDetail('/user', false);

    expect(await screen.findByText('保存草稿')).toBeInTheDocument();
    expect(screen.queryByText('添加盘盈行')).not.toBeInTheDocument();
  });
});
