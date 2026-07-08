import { describe, it, expect } from 'vitest';
import type { Asset, DeptTreeNode } from '@/types/api';
import {
  aggregateByCategory,
  collectSubtreeIds,
  enrichDeptStats,
  filterDeptStatsByIds,
  findDeptNode,
  sumDeptStats,
} from '@/utils/report';

const tree: DeptTreeNode[] = [
  {
    id: 10,
    parentId: 0,
    deptName: '信息工程学院',
    deptCode: 'INFO',
    path: '/10',
    children: [
      {
        id: 11,
        parentId: 10,
        deptName: '软件实验室',
        deptCode: 'SW',
        path: '/10/11',
        children: null,
      },
    ],
  },
];

describe('report utils', () => {
  it('aggregates assets by category', () => {
    const assets: Asset[] = [
      {
        id: 1,
        assetNo: 'A1',
        name: 'x',
        category: '设备',
        price: 100,
        purchaseTime: '2026-01-01',
        location: '101',
        departmentId: 11,
        userId: null,
        isShared: 0,
        status: 1,
      },
      {
        id: 2,
        assetNo: 'A2',
        name: 'y',
        category: '设备',
        price: 200,
        purchaseTime: '2026-01-01',
        location: '102',
        departmentId: 11,
        userId: null,
        isShared: 0,
        status: 1,
      },
    ];
    const stats = aggregateByCategory(assets);
    expect(stats).toHaveLength(1);
    expect(stats[0].count).toBe(2);
    expect(stats[0].totalValue).toBe(300);
  });

  it('collects subtree department ids', () => {
    const root = findDeptNode(tree, 10)!;
    expect(collectSubtreeIds(root)).toEqual([10, 11]);
  });

  it('enriches and filters dept stats', () => {
    const enriched = enrichDeptStats(
      [
        {
          departmentId: 10,
          totalCount: 5,
          inStockCount: 3,
          inUseCount: 2,
          totalValue: 1000,
        },
        {
          departmentId: 99,
          totalCount: 1,
          inStockCount: 1,
          inUseCount: 0,
          totalValue: 100,
        },
      ],
      new Map([[10, '信息工程学院']]),
    );
    const filtered = filterDeptStatsByIds(enriched, [10]);
    expect(filtered).toHaveLength(1);
    expect(filtered[0].departmentName).toBe('信息工程学院');
    expect(sumDeptStats(filtered).totalCount).toBe(5);
  });
});
