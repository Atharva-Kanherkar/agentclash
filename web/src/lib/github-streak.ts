/*
 * GitHub commit-streak data for the landing page.
 *
 * Fetches the last ~371 days of commits from a public repository, aggregates
 * them into a daily calendar, and tags each day with its most "landmark"
 * commit based on conventional-commit prefixes. Cached for one hour at the
 * Next.js fetch layer so we don't hammer the GitHub API on every render.
 */

const DAYS_BACK = 371;
const REVALIDATE_SECONDS = 60 * 60;

export type StreakLandmarkTier = "breaking" | "feat" | "fix" | "misc";

export type StreakLandmark = {
  message: string;
  sha: string;
  url: string;
  tier: StreakLandmarkTier;
};

export type StreakDay = {
  date: string; // YYYY-MM-DD (UTC)
  count: number;
  landmark?: StreakLandmark;
};

export type StreakData = {
  owner: string;
  repo: string;
  days: StreakDay[]; // chronological, length ~371
  totalCommits: number;
  activeDays: number;
  longestStreak: number;
  busiest: { date: string; count: number };
  rangeStart: string;
  rangeEnd: string;
};

type GhCommit = {
  sha: string;
  html_url: string;
  commit: {
    author: { date: string } | null;
    committer: { date: string } | null;
    message: string;
  };
};

function classify(message: string): StreakLandmarkTier | null {
  const subject = message.split("\n", 1)[0];
  if (/^[a-z]+(?:\([^)]+\))?!:/i.test(subject)) return "breaking";
  if (/BREAKING CHANGE/.test(message)) return "breaking";
  if (/^feat(?:\([^)]+\))?:/i.test(subject)) return "feat";
  if (/^fix(?:\([^)]+\))?:/i.test(subject)) return "fix";
  return null;
}

const TIER_PRIORITY: Record<StreakLandmarkTier, number> = {
  breaking: 3,
  feat: 2,
  fix: 1,
  misc: 0,
};

function emptyDayMap(rangeStart: Date, rangeEnd: Date): string[] {
  const dates: string[] = [];
  const cursor = new Date(
    Date.UTC(
      rangeStart.getUTCFullYear(),
      rangeStart.getUTCMonth(),
      rangeStart.getUTCDate(),
    ),
  );
  const end = new Date(
    Date.UTC(
      rangeEnd.getUTCFullYear(),
      rangeEnd.getUTCMonth(),
      rangeEnd.getUTCDate(),
    ),
  );
  while (cursor.getTime() <= end.getTime()) {
    dates.push(cursor.toISOString().slice(0, 10));
    cursor.setUTCDate(cursor.getUTCDate() + 1);
  }
  return dates;
}

export async function fetchRepoStreak(
  owner: string,
  repo: string,
): Promise<StreakData | null> {
  const now = new Date();
  const start = new Date(now);
  start.setUTCDate(start.getUTCDate() - (DAYS_BACK - 1));
  const sinceISO = new Date(
    Date.UTC(
      start.getUTCFullYear(),
      start.getUTCMonth(),
      start.getUTCDate(),
    ),
  ).toISOString();

  const headers: Record<string, string> = {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
  };
  const token = process.env.GITHUB_TOKEN;
  if (token) headers.Authorization = `Bearer ${token}`;

  const commits: GhCommit[] = [];
  for (let page = 1; page <= 30; page++) {
    const url = `https://api.github.com/repos/${owner}/${repo}/commits?per_page=100&since=${encodeURIComponent(sinceISO)}&page=${page}`;
    let res: Response;
    try {
      res = await fetch(url, {
        headers,
        next: { revalidate: REVALIDATE_SECONDS },
      });
    } catch {
      return null;
    }
    if (!res.ok) {
      if (page === 1) return null;
      break;
    }
    const batch = (await res.json()) as GhCommit[];
    if (!Array.isArray(batch) || batch.length === 0) break;
    commits.push(...batch);
    if (batch.length < 100) break;
  }

  const dates = emptyDayMap(start, now);
  const dayIndex = new Map<string, StreakDay>();
  for (const date of dates) dayIndex.set(date, { date, count: 0 });

  for (const c of commits) {
    const iso = c.commit.author?.date ?? c.commit.committer?.date;
    if (!iso) continue;
    const date = iso.slice(0, 10);
    const day = dayIndex.get(date);
    if (!day) continue;
    day.count++;

    const subject = c.commit.message.split("\n", 1)[0]?.trim() ?? "";
    const tier = classify(c.commit.message) ?? "misc";
    const candidate: StreakLandmark = {
      message: subject,
      sha: c.sha,
      url: c.html_url,
      tier,
    };
    if (
      !day.landmark ||
      TIER_PRIORITY[candidate.tier] > TIER_PRIORITY[day.landmark.tier]
    ) {
      day.landmark = candidate;
    }
  }

  const days = dates.map((d) => dayIndex.get(d)!);

  let longestStreak = 0;
  let currentStreak = 0;
  let totalCommits = 0;
  let activeDays = 0;
  let busiest = { date: days[0]?.date ?? "", count: 0 };
  for (const day of days) {
    totalCommits += day.count;
    if (day.count > 0) {
      activeDays++;
      currentStreak++;
      if (currentStreak > longestStreak) longestStreak = currentStreak;
    } else {
      currentStreak = 0;
    }
    if (day.count > busiest.count) {
      busiest = { date: day.date, count: day.count };
    }
  }

  // Strip landmark from days where the tier is "misc" — we only want the chip
  // to surface intentional, narratable commits.
  for (const day of days) {
    if (day.landmark && day.landmark.tier === "misc") {
      delete day.landmark;
    }
  }

  return {
    owner,
    repo,
    days,
    totalCommits,
    activeDays,
    longestStreak,
    busiest,
    rangeStart: days[0]?.date ?? "",
    rangeEnd: days[days.length - 1]?.date ?? "",
  };
}
