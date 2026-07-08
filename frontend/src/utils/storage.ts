const ACCESS_KEY = 'fams_access_token';
const REFRESH_KEY = 'fams_refresh_token';

export const storage = {
  getAccessToken: () => localStorage.getItem(ACCESS_KEY),
  setAccessToken: (t: string) => localStorage.setItem(ACCESS_KEY, t),
  getRefreshToken: () => localStorage.getItem(REFRESH_KEY),
  setRefreshToken: (t: string) => localStorage.setItem(REFRESH_KEY, t),
  clear: () => {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};
