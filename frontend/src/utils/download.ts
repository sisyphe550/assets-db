import { fetchWithAuth } from '@/utils/authFetch';

export async function downloadWithAuth(url: string, filename = 'export.csv') {
  const res = await fetchWithAuth(url);
  if (!res.ok) {
    throw new Error(`下载失败 (${res.status})`);
  }
  const blob = await res.blob();
  const objectUrl = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = objectUrl;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(objectUrl);
}
