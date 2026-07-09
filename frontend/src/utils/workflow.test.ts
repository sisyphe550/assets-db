import { describe, it, expect } from 'vitest';
import { canActOnWorkflow, filterAssetsForWorkflowType } from '@/utils/workflow';
import type { UserInfo, WorkflowRequest } from '@/types/api';

const baseRequest: WorkflowRequest = {
  id: 1,
  assetId: 10,
  requesterId: 3,
  departmentId: 15,
  type: 1,
  currentStage: 1,
  status: 1,
  reason: 'test',
  createdAt: '2026-07-01T00:00:00+08:00',
  updatedAt: '2026-07-01T00:00:00+08:00',
};

const collegeAdmin: UserInfo = {
  id: 2,
  username: 'admin_info',
  realName: '院管',
  roleLevel: 2,
  departmentId: 15,
  departmentName: '信息学院',
  status: 1,
};

const schoolAdmin: UserInfo = {
  ...collegeAdmin,
  id: 1,
  roleLevel: 1,
  username: 'admin_school',
};

describe('canActOnWorkflow', () => {
  it('allows college admin at stage 1', () => {
    expect(canActOnWorkflow(collegeAdmin, baseRequest)).toBe('approve_or_reject');
  });

  it('allows school admin at stage 2', () => {
    expect(
      canActOnWorkflow(schoolAdmin, { ...baseRequest, currentStage: 2 }),
    ).toBe('approve_or_reject');
  });

  it('denies when workflow is closed', () => {
    expect(canActOnWorkflow(collegeAdmin, { ...baseRequest, status: 2 })).toBeNull();
  });

  it('denies college admin at stage 2', () => {
    expect(
      canActOnWorkflow(collegeAdmin, { ...baseRequest, currentStage: 2 }),
    ).toBeNull();
  });

  it('denies college admin outside dept subtree', () => {
    expect(
      canActOnWorkflow(collegeAdmin, { ...baseRequest, departmentId: 20 }, [15, 103, 104]),
    ).toBeNull();
  });

  it('allows college admin inside dept subtree', () => {
    expect(canActOnWorkflow(collegeAdmin, baseRequest, [15, 103, 104])).toBe(
      'approve_or_reject',
    );
  });
});

describe('filterAssetsForWorkflowType', () => {
  const assets = [
    { id: 1, status: 1, userId: null },
    { id: 2, status: 2, userId: 5 },
    { id: 3, status: 2, userId: 3 },
    { id: 4, status: 3, userId: null },
  ];

  it('filters borrow type to in-stock', () => {
    expect(filterAssetsForWorkflowType(1, 3, assets)).toHaveLength(1);
  });

  it('filters return type to own borrowed assets', () => {
    expect(filterAssetsForWorkflowType(2, 3, assets)).toEqual([assets[2]]);
  });
});
