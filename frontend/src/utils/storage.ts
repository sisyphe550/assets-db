const ACCESS_KEY = 'fams_access_token';
const REFRESH_KEY = 'fams_refresh_token';

/** 清除旧版 localStorage token，避免多标签页登录互相覆盖 */
function clearLegacyLocalTokens() {
  try {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  } catch {
    // ignore quota / privacy mode errors
  }
}

clearLegacyLocalTokens();

export const storage = {
  getAccessToken: () => sessionStorage.getItem(ACCESS_KEY),
  setAccessToken: (t: string) => sessionStorage.setItem(ACCESS_KEY, t),
  getRefreshToken: () => sessionStorage.getItem(REFRESH_KEY),
  setRefreshToken: (t: string) => sessionStorage.setItem(REFRESH_KEY, t),
  clear: () => {
    sessionStorage.removeItem(ACCESS_KEY);
    sessionStorage.removeItem(REFRESH_KEY);
  },
};
