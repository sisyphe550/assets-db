import { ConfigProvider } from 'antd';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import zhCN from 'antd/locale/zh_CN';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import InventoryTaskItemConfigurator from './InventoryTaskItemConfigurator';

const mocks = vi.hoisted(() => ({
  updateTaskItems: vi.fn(),
  publishTask: vi.fn(),
  refetch: vi.fn(),
}));

vi.mock('@/store/api/inventoryApi', () => ({
  useGetTaskItemsQuery: () => ({
    data: {
      list: [
        {
          assetId: 1,
          assetNo: 'EQUIP-2026-0001',
          name: '激光切割机',
          bookLocation: '一号实验楼101',
        },
        {
          assetId: 2,
          assetNo: 'EQUIP-2026-0002',
          name: '3D打印机',
          bookLocation: '一号实验楼102',
        },
      ],
      available: [
        {
          assetId: 1,
          assetNo: 'EQUIP-2026-0001',
          name: '激光切割机',
          bookLocation: '一号实验楼101',
        },
        {
          assetId: 2,
          assetNo: 'EQUIP-2026-0002',
          name: '3D打印机',
          bookLocation: '一号实验楼102',
        },
      ],
      total: 2,
    },
    isLoading: false,
    refetch: mocks.refetch,
  }),
  useUpdateTaskItemsMutation: () => [mocks.updateTaskItems, { isLoading: false }],
  usePublishTaskMutation: () => [mocks.publishTask, { isLoading: false }],
}));

describe('InventoryTaskItemConfigurator', () => {
  beforeEach(() => {
    mocks.updateTaskItems.mockReset();
    mocks.updateTaskItems.mockReturnValue({ unwrap: () => Promise.resolve() });
    mocks.publishTask.mockReset();
    mocks.publishTask.mockReturnValue({ unwrap: () => Promise.resolve() });
    mocks.refetch.mockReset();
  });

  it('renders selected asset entries and deletes by saving remaining asset ids', async () => {
    render(
      <ConfigProvider locale={zhCN}>
        <InventoryTaskItemConfigurator taskId={11} />
      </ConfigProvider>,
    );

    expect(screen.getByText('EQUIP-2026-0001')).toBeInTheDocument();
    expect(screen.getByText('激光切割机')).toBeInTheDocument();

    fireEvent.click(screen.getAllByText('删除')[0]);

    await waitFor(() => {
      expect(mocks.updateTaskItems).toHaveBeenCalledWith({
        taskId: 11,
        assetIds: [2],
      });
    });
  });
});
