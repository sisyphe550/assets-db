export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  tokenType: string;
}

export interface UserInfo {
  id: number;
  username: string;
  realName: string;
  roleLevel: 1 | 2 | 3;
  departmentId: number;
  departmentName: string;
  status: number;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface Asset {
  id: number;
  assetNo: string;
  name: string;
  category: string;
  price: number;
  purchaseTime: string;
  location: string;
  departmentId: number;
  userId: number | null;
  isShared: number;
  status: 1 | 2 | 3 | 4;
}

export interface CreateAssetReq {
  assetNo: string;
  name: string;
  category: string;
  price: number;
  purchaseTime: string;
  location: string;
  departmentId: number;
  isShared: number;
}

export type UpdateAssetReq = Partial<
  Pick<CreateAssetReq, 'name' | 'category' | 'location' | 'departmentId' | 'isShared'>
>;

export interface AssetListParams {
  page?: number;
  pageSize?: number;
  category?: string;
  status?: number;
  keyword?: string;
  scope?: 'my';
  userId?: string | number;
}

export interface DeptTreeNode {
  id: number;
  parentId: number;
  deptName: string;
  deptCode: string;
  path: string;
  children: DeptTreeNode[] | null;
}

export interface WorkflowRequest {
  id: number;
  assetId: number;
  requesterId: number;
  departmentId: number;
  type: 1 | 2 | 3 | 4;
  currentStage: 1 | 2 | 3;
  status: 1 | 2 | 3;
  reason: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorkflowLog {
  id: number;
  requestId: number;
  operatorId: number;
  action: string;
  comment: string;
  operateTime: string;
}

export interface WorkflowDetail {
  request: WorkflowRequest;
  logs: WorkflowLog[];
}

export interface CreateWorkflowReq {
  assetId: number;
  type: 1 | 2 | 3 | 4;
  reason: string;
}

export interface WorkflowListParams {
  page?: number;
  pageSize?: number;
  scope?: 'my' | 'todo' | 'done' | 'all';
  type?: number;
  status?: number;
  assetId?: number;
}

export interface ApproveWorkflowReq {
  comment?: string;
}
