import type { AppMenuItem } from '@/config/menu';

/** 按路径最长前缀匹配侧边栏选中项 */
export function matchMenuSelectedKey(pathname: string, items: AppMenuItem[]): string {
  const keys = items
    .map((m) => (m && typeof m === 'object' && 'key' in m ? String(m.key) : ''))
    .filter((k) => k && pathname.startsWith(k))
    .sort((a, b) => b.length - a.length);
  return keys[0] ?? pathname;
}
