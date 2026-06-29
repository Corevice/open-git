export const BRANDING = {
  appName: "OpenGit",
  logoSrc: "/brand/logo.svg",
  faviconSrc: "/brand/favicon.ico",
  primaryColor: "#1f6feb",
  sourceUrl: "https://github.com/open-git/open-git",
  licenseName: "Apache-2.0",
} as const;

export type Branding = typeof BRANDING;
