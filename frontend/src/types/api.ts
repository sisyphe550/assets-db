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
