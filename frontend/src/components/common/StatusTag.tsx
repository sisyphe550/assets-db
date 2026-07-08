import { Tag } from 'antd';
import { ASSET_STATUS_MAP } from '@/utils/constants';

type StatusTagType = 'asset';

interface StatusTagProps {
  type: StatusTagType;
  value: number;
}

export default function StatusTag({ type, value }: StatusTagProps) {
  if (type === 'asset') {
    const meta = ASSET_STATUS_MAP[value] ?? { label: `未知(${value})`, color: 'default' };
    return <Tag color={meta.color}>{meta.label}</Tag>;
  }
  return <Tag>{value}</Tag>;
}
