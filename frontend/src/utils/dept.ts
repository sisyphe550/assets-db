import type { DataNode } from 'antd/es/tree';
import type { DeptTreeNode } from '@/types/api';

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

export function filterDeptTreeByIds(nodes: DeptTreeNode[], allowedIds: Set<number>): DeptTreeNode[] {
  const walk = (list: DeptTreeNode[]): DeptTreeNode[] => {
    const result: DeptTreeNode[] = [];
    for (const node of list) {
      const childNodes = node.children?.length ? walk(node.children) : [];
      if (allowedIds.has(node.id) || childNodes.length > 0) {
        result.push({
          ...node,
          children: childNodes.length ? childNodes : null,
        });
      }
    }
    return result;
  };
  return walk(nodes);
}

export function toAntdTreeData(nodes: DeptTreeNode[]): DataNode[] {
  return nodes.map((node) => ({
    key: String(node.id),
    title: node.deptName,
    children: node.children?.length ? toAntdTreeData(node.children) : undefined,
  }));
}

export interface DeptTreeSelectNode {
  title: string;
  value: number;
  children?: DeptTreeSelectNode[];
}

export function toTreeSelectData(nodes: DeptTreeNode[]): DeptTreeSelectNode[] {
  return nodes.map((node) => ({
    title: node.deptName,
    value: node.id,
    children: node.children?.length ? toTreeSelectData(node.children) : undefined,
  }));
}

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
