import dayjs from 'dayjs';

export const formatDate = (iso: string) => dayjs(iso).format('YYYY-MM-DD');
export const formatDateTime = (iso: string) => dayjs(iso).format('YYYY-MM-DD HH:mm');
export const formatPrice = (n: number) =>
  `¥${n.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`;
