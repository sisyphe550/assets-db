import type { UserInfo, WorkflowRequest } from '@/types/api';
import { WORKFLOW_STAGE_MAP } from '@/utils/constants';

export type WorkflowAction = 'approve_or_reject';

/** 结合 status 展示阶段：驳回/通过后不再显示「待初审」等进行中标签 */
export function getWorkflowStageLabel(status: number, currentStage: number): string {
  if (status === 2) return WORKFLOW_STAGE_MAP[3];
  if (status === 3) return '已完结';
  return WORKFLOW_STAGE_MAP[currentStage] ?? `阶段${currentStage}`;
}

export function canActOnWorkflow(
  user: UserInfo,
  request: WorkflowRequest,
  deptSubtreeIds?: number[] | null,
): WorkflowAction | null {
  if (request.status !== 1) return null;
  if (user.roleLevel === 2 && request.currentStage === 1) {
    if (deptSubtreeIds?.length && !deptSubtreeIds.includes(request.departmentId)) {
      return null;
    }
    return 'approve_or_reject';
  }
  if (user.roleLevel === 1 && request.currentStage === 2) return 'approve_or_reject';
  return null;
}

export function getAssetQueryForWorkflowType(
  type: number,
  _userId: number,
): { status?: number; scope?: 'my'; userId?: string } {
  switch (type) {
    case 1:
      return { status: 1 };
    case 2:
      return { status: 2, scope: 'my' };
    case 3:
      return {};
    case 4:
      return {};
    default:
      return {};
  }
}

export function filterAssetsForWorkflowType<T extends { status: number; userId: number | null }>(
  type: number,
  userId: number,
  assets: T[],
): T[] {
  switch (type) {
    case 1:
      return assets.filter((a) => a.status === 1);
    case 2:
      return assets.filter((a) => a.status === 2 && a.userId === userId);
    case 3:
      return assets.filter((a) => a.status === 1 || a.status === 2);
    case 4:
      return assets.filter((a) => a.status === 1 || a.status === 3);
    default:
      return assets;
  }
}
