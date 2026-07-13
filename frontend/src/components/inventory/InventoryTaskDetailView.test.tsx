import { ConfigProvider } from 'antd';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import zhCN from 'antd/locale/zh_CN';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import InventoryTaskDetailView from './InventoryTaskDetailView';

const mocks = vi.hoisted(() => ({
  compareTask: vi.fn(),
}));

vi.mock('@/store/hooks', () => ({
  useAppSelector: () => ({ id: 10001, roleLevel: 1 }),
}));

vi.mock('@/components/inventory/InventoryConflictPanel', () => ({
  default: ({ taskId }: { taskId: number }) => <div>裁决面板-{taskId}</div>,
}));

vi.mock('@/store/api/inventoryApi', () => ({
  useGetTaskQuery: () => ({
    data: {
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
    },
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
  useGetExpectedAssetsQuery: () => ({ data: undefined, isLoading: false }),
  useGetDraftsQuery: () => ({ data: undefined, isLoading: false, refetch: vi.fn() }),
  useGetRecordsQuery: () => ({ data: undefined }),
  useSubmitRecordsMutation: () => [vi.fn(), { isLoading: false }],
  useArchiveTaskMutation: () => [vi.fn(), { isLoading: false }],
  useCompareTaskMutation: () => [mocks.compareTask, { isLoading: false }],
}));

function renderDetail() {
  return render(
    <ConfigProvider locale={zhCN}>
      <MemoryRouter initialEntries={['/admin/inventory/tasks/11']}>
        <Routes>
          <Route
            path="/admin/inventory/tasks/:id"
            element={<InventoryTaskDetailView basePath="/admin" showArchive />}
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
  });

  it('uses the conflict endpoint as source of truth when task count is stale', () => {
    renderDetail();

    expect(screen.getByText('裁决面板-11')).toBeInTheDocument();
    expect(mocks.compareTask).not.toHaveBeenCalled();
  });
});
