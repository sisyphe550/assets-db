import { Tag } from 'antd';
import {
  ASSET_STATUS_MAP,
  INVENTORY_DIFF_MAP,
  INVENTORY_STATUS_MAP,
  WORKFLOW_STAGE_MAP,
  WORKFLOW_STATUS_MAP,
  WORKFLOW_TYPE_MAP,
} from '@/utils/constants';

type StatusTagType =
  | 'asset'
  | 'workflow'
  | 'workflowStage'
  | 'workflowType'
  | 'inventory'
  | 'inventoryDiff';

interface StatusTagProps {
  type: StatusTagType;
  value: number;
}

export default function StatusTag({ type, value }: StatusTagProps) {
  if (type === 'asset') {
    const meta = ASSET_STATUS_MAP[value] ?? { label: `未知(${value})`, color: 'default' };
    return <Tag color={meta.color}>{meta.label}</Tag>;
  }
  if (type === 'workflow') {
    const meta = WORKFLOW_STATUS_MAP[value] ?? { label: `未知(${value})`, color: 'default' };
    return <Tag color={meta.color}>{meta.label}</Tag>;
  }
  if (type === 'workflowStage') {
    return <Tag>{WORKFLOW_STAGE_MAP[value] ?? `阶段${value}`}</Tag>;
  }
  if (type === 'workflowType') {
    return <Tag color="blue">{WORKFLOW_TYPE_MAP[value] ?? `类型${value}`}</Tag>;
  }
  if (type === 'inventory') {
    const meta = INVENTORY_STATUS_MAP[value] ?? { label: `未知(${value})`, color: 'default' };
    return <Tag color={meta.color}>{meta.label}</Tag>;
  }
  if (type === 'inventoryDiff') {
    const meta = INVENTORY_DIFF_MAP[value] ?? { label: `未知(${value})`, color: 'default' };
    return <Tag color={meta.color}>{meta.label}</Tag>;
  }
  return <Tag>{value}</Tag>;
}
