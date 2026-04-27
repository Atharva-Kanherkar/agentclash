import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { fetchRepoStreak } from "@/lib/github-streak";
import HomePage from "./landing";

export default async function RootPage() {
  const { user } = await withAuth();
  if (user) redirect("/dashboard");
  const streak = await fetchRepoStreak("agentclash", "agentclash");
  return <HomePage streak={streak} />;
}
