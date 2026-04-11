"use client";

import { createContext, useContext, useCallback } from "react";
import type { UserMeResponse } from "@/lib/api/types";

interface AuthContextValue {
  /** The authenticated user's profile, including orgs and workspaces. */
  user: UserMeResponse;
  /** Navigate to the sign-out route to destroy the session. */
  signOut: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/**
 * Provides auth context to client components within the dashboard.
 * Must receive pre-fetched user data from the server layout.
 */
export function AuthProvider({
  user,
  children,
}: {
  user: UserMeResponse;
  children: React.ReactNode;
}) {
  const signOut = useCallback(() => {
    window.location.href = "/auth/sign-out";
  }, []);

  return (
    <AuthContext.Provider value={{ user, signOut }}>
      {children}
    </AuthContext.Provider>
  );
}

/**
 * Access the authenticated user and sign-out function.
 * Must be used within a dashboard route (under AuthProvider).
 */
export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider (dashboard layout)");
  }
  return context;
}
