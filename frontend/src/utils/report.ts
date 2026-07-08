import type { Asset, DeptStatItem, DeptTreeNode, CategoryStatItem } from '@/types/api';

export const CHART_COLORS = ['#1677FF', '#52c41a', '#faad14', '#ff4d4f', '#722ed1'];

export function flattenDeptTree(
  nodes: DeptTreeNode[],
  acc: { id: number; name: string }[] = [],
): { id: number; name: string }[] {
  for (const node of nodes) {
    acc.push({ id: node.id, name: node.deptName });
    if (node.children?.length) {
      flattenDeptTree(node.children, acc);
    }
  }
  return acc;
}

export function findDeptNode(nodes: DeptTreeNode[], id: number): DeptTreeNode | null {
  for (const node of nodes) {
    if (node.id === id) return node;
    if (node.children?.length) {
      const found = findDeptNode(node.children, id);
      if (found) return found;
    }
  }
  return null;
}

export function collectSubtreeIds(root: DeptTreeNode): number[] {
  const ids = [root.id];
  for (const child of root.children ?? []) {
    ids.push(...collectSubtreeIds(child));
  }
  return ids;
}

export function enrichDeptStats(
  items: Omit<DeptStatItem, 'departmentName'>[],
  deptMap: Map<number, string>,
): DeptStatItem[] {
  return items.map((item) => ({
    ...item,
    departmentName: deptMap.get(item.departmentId) ?? `部门#${item.departmentId}`,
  }));
}

export function filterDeptStatsByIds(items: DeptStatItem[], deptIds: number[]): DeptStatItem[] {
  const allowed = new Set(deptIds);
  return items.filter((item) => allowed.has(item.departmentId));
}

export function aggregateByCategory(assets: Asset[]): CategoryStatItem[] {
  const map = new Map<string, CategoryStatItem>();
  for (const asset of assets) {
    const existing = map.get(asset.category);
    if (existing) {
      existing.count += 1;
      existing.totalValue += asset.price;
    } else {
      map.set(asset.category, {
        category: asset.category,
        count: 1,
        totalValue: asset.price,
      });
    }
  }
  return [...map.values()].sort((a, b) => b.count - a.count);
}

export function sumDeptStats(items: DeptStatItem[]) {
  return items.reduce(
    (acc, item) => ({
      totalCount: acc.totalCount + item.totalCount,
      inStockCount: acc.inStockCount + item.inStockCount,
      inUseCount: acc.inUseCount + item.inUseCount,
      totalValue: acc.totalValue + item.totalValue,
    }),
    { totalCount: 0, inStockCount: 0, inUseCount: 0, totalValue: 0 },
  );
}

export function formatWan(value: number) {
  if (value >= 10000) {
    return `${(value / 10000).toFixed(1)}万`;
  }
  return value.toLocaleString('zh-CN');
}
