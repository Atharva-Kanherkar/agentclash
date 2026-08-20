import { Calendar } from "lucide-react";
import { TrackedLink } from "@/components/analytics/tracked-cta";

const DEMO_LINK = "atharva-kanherkar-epgztu/agentclash-demo";
const DEMO_BUTTON_CONFIG = JSON.stringify({ layout: "month_view" });

type Props = {
  className?: string;
  label?: string;
  ctaId?: string;
  placement?: string;
};

export function DemoButton({
  className = "inline-flex items-center justify-center gap-2 rounded-md bg-white px-6 py-3 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors",
  label = "Book a demo",
  ctaId = "marketing.demo-button.demo",
  placement = "demo_button",
}: Props) {
  return (
    <TrackedLink
      href={`https://cal.com/${DEMO_LINK}`}
      ctaId={ctaId}
      intent="demo"
      placement={placement}
      data-cal-link={DEMO_LINK}
      data-cal-config={DEMO_BUTTON_CONFIG}
      className={className}
    >
      <Calendar className="size-4" />
      {label}
    </TrackedLink>
  );
}
