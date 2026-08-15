import { TryCliLandingClient } from "@/components/try-cli/landing-client";
import { getPublicTryCliDemos } from "@/lib/public-page-data";

export default async function TryCliPage() {
  const demos = await getPublicTryCliDemos();
  return <TryCliLandingClient initialDemos={demos} />;
}
