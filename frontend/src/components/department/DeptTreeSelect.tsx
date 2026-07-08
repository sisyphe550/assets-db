import { useMemo } from 'react';
import { TreeSelect } from 'antd';
import { useGetDeptTreeQuery } from '@/store/api/userApi';
import {
  collectSubtreeIds,
  filterDeptTreeByIds,
  findDeptNode,
  toTreeSelectData,
} from '@/utils/dept';

interface DeptTreeSelectProps {
  value?: number;
  onChange?: (id: number) => void;
  disabled?: boolean;
  restrictSubtree?: number;
}

export default function DeptTreeSelect({
  value,
  onChange,
  disabled,
  restrictSubtree,
}: DeptTreeSelectProps) {
  const { data: deptTree, isLoading } = useGetDeptTreeQuery();

  const treeData = useMemo(() => {
    if (!deptTree?.length) return [];
    if (!restrictSubtree) {
      return toTreeSelectData(deptTree);
    }
    const root = findDeptNode(deptTree, restrictSubtree);
    if (!root) return [];
    const allowed = new Set(collectSubtreeIds(root));
    return toTreeSelectData(filterDeptTreeByIds(deptTree, allowed));
  }, [deptTree, restrictSubtree]);

  return (
    <TreeSelect
      value={value}
      onChange={onChange}
      disabled={disabled}
      loading={isLoading}
      treeData={treeData}
      placeholder="请选择部门"
      showSearch
      treeDefaultExpandAll
      treeNodeFilterProp="title"
      style={{ width: '100%' }}
    />
  );
}
