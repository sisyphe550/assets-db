export const ROLE_HOME: Record<number, string> = {
  1: '/admin/dashboard',
  2: '/college/dashboard',
  3: '/user/assets',
};

export const ROLE_MAP: Record<number, string> = {
  1: '校级管理员',
  2: '学院管理员',
  3: '普通师生',
};

export const ASSET_STATUS_MAP: Record<number, { label: string; color: string }> = {
  1: { label: '在库', color: 'green' },
  2: { label: '领用中', color: 'blue' },
  3: { label: '维修中', color: 'orange' },
  4: { label: '已报废', color: 'default' },
};

export const ASSET_CATEGORIES = ['设备', '家具', '实验器材', '办公用品', '其他'] as const;

export const WORKFLOW_TYPE_MAP: Record<number, string> = {
  1: '领用',
  2: '归还',
  3: '报修',
  4: '报废',
};

export const WORKFLOW_STATUS_MAP: Record<number, { label: string; color: string }> = {
  1: { label: '审批中', color: 'processing' },
  2: { label: '已通过', color: 'success' },
  3: { label: '已驳回', color: 'error' },
};

export const WORKFLOW_STAGE_MAP: Record<number, string> = {
  1: '待院级初审',
  2: '待校级复审',
  3: '已归档',
};
