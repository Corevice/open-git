import "@/lib/env";
import type { Metadata } from "next";
import { AuthProvider } from "@/components/providers/auth-provider";
import { AuthProvider as TokenAuthProvider } from "@/lib/auth";
import { QueryProvider } from "@/components/providers/query-provider";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { ToastProvider } from "@/components/ui/toast";
import { BRANDING } from "@/lib/branding";
import "./globals.css";

export const metadata: Metadata = {
  title: BRANDING.appName,
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ja" suppressHydrationWarning>
      <body>
        <ThemeProvider>
          <QueryProvider>
            <ToastProvider>
              <AuthProvider>
                <TokenAuthProvider>{children}</TokenAuthProvider>
              </AuthProvider>
            </ToastProvider>
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
