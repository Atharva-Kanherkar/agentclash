"use server";

import { redirect } from "next/navigation";
import { createDevSession } from "@/lib/auth/session";

export async function devSignIn(formData: FormData) {
  const userId = formData.get("userId");
  const email = formData.get("email");
  const displayName = formData.get("displayName");
  const orgMemberships = formData.get("orgMemberships");
  const workspaceMemberships = formData.get("workspaceMemberships");

  if (typeof userId !== "string" || !userId.trim()) {
    throw new Error("User ID is required");
  }
  if (typeof email !== "string" || !email.trim()) {
    throw new Error("Email is required");
  }

  await createDevSession({
    userId: userId.trim(),
    email: email.trim(),
    displayName:
      typeof displayName === "string" && displayName.trim()
        ? displayName.trim()
        : email.split("@")[0],
    orgMemberships:
      typeof orgMemberships === "string" ? orgMemberships.trim() : "",
    workspaceMemberships:
      typeof workspaceMemberships === "string"
        ? workspaceMemberships.trim()
        : "",
  });

  redirect("/dashboard");
}
