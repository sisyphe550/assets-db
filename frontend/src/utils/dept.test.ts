import { describe, it, expect } from 'vitest';
import type { DeptTreeNode } from '@/types/api';
import {
  collectSubtreeIds,
  filterDeptTreeByIds,
  findDeptNode,
  toAntdTreeData,
  toTreeSelectData,
} from '@/utils/dept';

const tree: DeptTreeNode[] = [
  {
    id: 1,
    parentId: 0,
    deptName: '本校',
    deptCode: 'ROOT',
    path: '/1/',
    children: [
      {
        id: 10,
        parentId: 1,
        deptName: '信息工程学院',
        deptCode: 'INFO',
        path: '/1/10/',
        children: [
          {
            id: 11,
            parentId: 10,
            deptName: '软件实验室',
            deptCode: 'SW',
            path: '/1/10/11/',
            children: null,
          },
        ],
      },
    ],
  },
];

describe('dept utils', () => {
  it('finds node and collects subtree ids', () => {
    const node = findDeptNode(tree, 10);
    expect(node?.deptName).toBe('信息工程学院');
    expect(collectSubtreeIds(node!)).toEqual([10, 11]);
  });

  it('filters tree by allowed ids', () => {
    const filtered = filterDeptTreeByIds(tree, new Set([10, 11]));
    expect(filtered).toHaveLength(1);
    expect(filtered[0].id).toBe(1);
    expect(filtered[0].children?.[0].id).toBe(10);
    expect(filtered[0].children?.[0].children?.[0].id).toBe(11);
  });

  it('converts to antd tree and tree select data', () => {
    expect(toAntdTreeData(tree)[0].key).toBe('1');
    expect(toTreeSelectData(tree)[0].value).toBe(1);
    expect(toTreeSelectData(tree)[0].children?.[0].title).toBe('信息工程学院');
  });
});
