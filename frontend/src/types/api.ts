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

export interface CreateDeptReq {
  parentId: number;
  deptName: string;
  deptCode: string;
  sortOrder?: number;
}

export interface CreateUserReq {
  username: string;
  password: string;
  realName: string;
  roleLevel: 1 | 2 | 3;
  departmentId: number;
}

export interface UpdateUserStatusReq {
  status: 0 | 1;
}

export interface CollegeSubtreeResponse {
  deptIds: number[];
}

export interface WorkflowRequest {
  id: number;
  assetId: number;
  requesterId: number;
  requesterName?: string;
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

export interface InventoryTask {
  id: number;
  taskName: string;
  scopeDeptId: number;
  creatorId: number;
  startTime: string;
  endTime: string;
  status: 1 | 2 | 3;
  assigneeIds: number[];
  expectedAssetCount?: number;
  submittedCount?: number;
  pendingConflictCount?: number;
}

export interface InventoryConflictCandidate {
  operatorId: number;
  operatorName?: string;
  actualLocation: string;
  notes: string;
  foundName: string;
  updatedAt: string;
}

export interface InventoryConflict {
  assetNo: string;
  assetId?: number;
  name?: string;
  bookLocation?: string;
  candidates: InventoryConflictCandidate[];
}

export interface ResolveConflictReq {
  source: 'assignee' | 'custom';
  operatorId?: number;
  actualLocation?: string;
  notes?: string;
}

export interface CreateInventoryTaskReq {
  taskName: string;
  scopeDeptId: number;
  startTime: string;
  endTime: string;
  assigneeIds: number[];
}

export interface ExpectedAsset {
  assetId: number;
  assetNo: string;
  name: string;
  bookLocation: string;
  expectedUpdatedAt?: string | null;
}

export interface InventoryDraft {
  assetNo: string;
  operatorId?: number;
  modifiedCells: Record<string, string>;
  updatedAt: string;
}

export interface SubmitItem {
  assetNo: string;
  modifiedCells: Record<string, string>;
  expectedUpdatedAt: string | null;
}

export interface SubmitResult {
  success: string[];
  conflicts: { assetNo: string; code: number; message: string }[];
  failures: { assetNo: string; code: number; message: string }[];
}

export interface InventoryRecord {
  assetNo: string;
  name: string;
  bookLocation: string;
  actualLocation: string;
  diffStatus: 0 | 1 | 2 | 3;
}

export interface UserListItem {
  id: number;
  username: string;
  realName: string;
  roleLevel: number;
  departmentId: number;
  departmentName: string;
  status: number;
}

export interface UserListParams {
  page?: number;
  pageSize?: number;
  departmentId?: number;
  roleLevel?: number;
  keyword?: string;
}

export interface DeptStatItem {
  departmentId: number;
  departmentName?: string;
  totalCount: number;
  inStockCount: number;
  inUseCount: number;
  totalValue: number;
}

export interface DeptStatResponse {
  items: Omit<DeptStatItem, 'departmentName'>[];
}

export interface CategoryStatItem {
  category: string;
  count: number;
  totalValue: number;
}

export interface CategoryStatResponse {
  items: CategoryStatItem[];
}

export interface InventoryDiffReport {
  match: number;
  surplus: number;
  loss: number;
}

export interface ExportJob {
  jobId: number;
  status: 0 | 1 | 2 | 3;
  downloadUrl?: string | null;
  errorMessage?: string | null;
}

export interface CreateExportReq {
  exportType: 'asset_list' | 'inventory_diff' | 'workflow_log';
  params?: Record<string, unknown>;
}
