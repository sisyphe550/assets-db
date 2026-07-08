import type { UserInfo, WorkflowRequest } from '@/types/api';

export type WorkflowAction = 'approve_or_reject';

export function canActOnWorkflow(user: UserInfo, request: WorkflowRequest): WorkflowAction | null {
  if (request.status !== 1) return null;
  if (user.roleLevel === 2 && request.currentStage === 1) return 'approve_or_reject';
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
