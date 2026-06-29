import { cookies } from "next/headers";

import { API_TOKEN_KEY } from "./api";
import { env } from "./env";

export type SessionUser = {
  id?: number | string;
  login?: string;
  name?: string;
  role?: string;
};

export type ServerSession = {
  token: string;
  user: SessionUser | null;
};

/**
 * Reads the auth token from the request cookies and resolves the current
 * viewer from the API. Returns `null` when there is no authenticated session.
 *
 * Reading cookies opts the consuming route into dynamic rendering, so the
 * upstream API is never called during static prerendering / build.
 */
export async function getServerSession(): Promise<ServerSession | null> {
  const cookieStore = await cookies();
  const token = cookieStore.get(API_TOKEN_KEY)?.value;

  if (!token) {
    return null;
  }

  try {
    const response = await fetch(
      `${env.NEXT_PUBLIC_API_BASE_URL.replace(/\/$/, "")}/api/v3/user`,
      {
        headers: {
          Accept: "application/json",
          Authorization: `Bearer ${token}`,
        },
        cache: "no-store",
      },
    );

    if (!response.ok) {
      return { token, user: null };
    }

    const user = (await response.json()) as SessionUser;
    return { token, user };
  } catch {
    return { token, user: null };
  }
}
