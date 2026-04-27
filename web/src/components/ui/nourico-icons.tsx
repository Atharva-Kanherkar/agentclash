import * as React from "react";

export type LucideIcon = React.ForwardRefExoticComponent<
  Omit<NouricoIconProps, "ref"> & React.RefAttributes<SVGSVGElement>
>;

export type NouricoIconProps = React.SVGProps<SVGSVGElement> & {
  absoluteStrokeWidth?: boolean;
  size?: string | number;
  strokeWidth?: string | number;
};

const NouricoIcon = React.forwardRef<SVGSVGElement, NouricoIconProps>(
  (
    {
      absoluteStrokeWidth,
      children,
      color = "currentColor",
      fill = "none",
      height,
      size = 24,
      stroke = "currentColor",
      strokeWidth = 1.5,
      width,
      ...props
    },
    ref,
  ) => {
    void absoluteStrokeWidth;

    return (
      <svg
        ref={ref}
        width={width ?? size}
        height={height ?? size}
        viewBox="0 0 24 24"
        fill={fill}
        stroke={stroke === "currentColor" ? color : stroke}
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={strokeWidth}
        role="presentation"
        aria-hidden={props["aria-label"] ? undefined : true}
        {...props}
      >
        <path d="M3.361 16.104c-.481-3.393-.481-4.815 0-8.208C3.595 6.248 5.406 5 7.563 5h.217c1.378 0 2.655.721 3.368 1.9l2.49 4.122c.505.836 1.79.475 1.786-.501l-.016-4.489C15.406 5.463 15.867 5 16.437 5c2.157 0 3.968 1.248 4.202 2.896.374 2.639.458 3.726.249 6.016a2.92 2.92 0 0 1-.721 1.661l-1.571 1.791c-1.633 1.86-4.572 1.719-6.018-.29l-2.229-3.094c-.565-.785-1.804-.391-1.812.576l-.029 3.507A.946.946 0 0 1 7.563 19c-2.157 0-3.968-1.248-4.202-2.896Z" />
        {children}
      </svg>
    );
  },
);

NouricoIcon.displayName = "NouricoIcon";

export { NouricoIcon };
export default NouricoIcon;

export const Activity = NouricoIcon;
export const AlertCircle = NouricoIcon;
export const AlertOctagon = NouricoIcon;
export const AlertTriangle = NouricoIcon;
export const ArrowDown = NouricoIcon;
export const ArrowDownRight = NouricoIcon;
export const ArrowLeft = NouricoIcon;
export const ArrowRight = NouricoIcon;
export const ArrowRightCircle = NouricoIcon;
export const ArrowUp = NouricoIcon;
export const ArrowUpDown = NouricoIcon;
export const ArrowUpRight = NouricoIcon;
export const AudioLines = NouricoIcon;
export const BarChart3 = NouricoIcon;
export const BookOpen = NouricoIcon;
export const Bot = NouricoIcon;
export const BrainCircuit = NouricoIcon;
export const Calendar = NouricoIcon;
export const Check = NouricoIcon;
export const CheckCircle = NouricoIcon;
export const CheckCircle2 = NouricoIcon;
export const CheckIcon = NouricoIcon;
export const ChevronDown = NouricoIcon;
export const ChevronDownIcon = NouricoIcon;
export const ChevronLeft = NouricoIcon;
export const ChevronRight = NouricoIcon;
export const ChevronRightIcon = NouricoIcon;
export const ChevronUpIcon = NouricoIcon;
export const ChevronsUpDown = NouricoIcon;
export const CircleCheckIcon = NouricoIcon;
export const CircleDot = NouricoIcon;
export const Clipboard = NouricoIcon;
export const ClipboardCheck = NouricoIcon;
export const ClipboardList = NouricoIcon;
export const Clock = NouricoIcon;
export const Code2 = NouricoIcon;
export const Cog = NouricoIcon;
export const Coins = NouricoIcon;
export const Command = NouricoIcon;
export const Copy = NouricoIcon;
export const Database = NouricoIcon;
export const DollarSign = NouricoIcon;
export const Download = NouricoIcon;
export const ExternalLink = NouricoIcon;
export const FileArchive = NouricoIcon;
export const FileCode = NouricoIcon;
export const FileUp = NouricoIcon;
export const Flag = NouricoIcon;
export const FlaskConical = NouricoIcon;
export const Gauge = NouricoIcon;
export const GitCompare = NouricoIcon;
export const Github = NouricoIcon;
export const Hash = NouricoIcon;
export const History = NouricoIcon;
export const Inbox = NouricoIcon;
export const Info = NouricoIcon;
export const InfoIcon = NouricoIcon;
export const Key = NouricoIcon;
export const Layers = NouricoIcon;
export const ListChecks = NouricoIcon;
export const ListTree = NouricoIcon;
export const Loader2 = NouricoIcon;
export const Loader2Icon = NouricoIcon;
export const Lock = NouricoIcon;
export const LogIn = NouricoIcon;
export const LogOut = NouricoIcon;
export const Map = NouricoIcon;
export const Maximize2 = NouricoIcon;
export const MessageSquare = NouricoIcon;
export const MessageSquareText = NouricoIcon;
export const Minimize2 = NouricoIcon;
export const Minus = NouricoIcon;
export const MoreHorizontal = NouricoIcon;
export const MoreHorizontalIcon = NouricoIcon;
export const NotebookPen = NouricoIcon;
export const OctagonXIcon = NouricoIcon;
export const Package = NouricoIcon;
export const PackageOpen = NouricoIcon;
export const PanelLeft = NouricoIcon;
export const PanelLeftClose = NouricoIcon;
export const Pencil = NouricoIcon;
export const Play = NouricoIcon;
export const PlayCircle = NouricoIcon;
export const Plus = NouricoIcon;
export const Quote = NouricoIcon;
export const Radio = NouricoIcon;
export const Rocket = NouricoIcon;
export const Save = NouricoIcon;
export const Search = NouricoIcon;
export const Settings2 = NouricoIcon;
export const Share2 = NouricoIcon;
export const Shield = NouricoIcon;
export const ShieldAlert = NouricoIcon;
export const ShieldCheck = NouricoIcon;
export const ShieldQuestion = NouricoIcon;
export const ShieldX = NouricoIcon;
export const Sigma = NouricoIcon;
export const Sparkles = NouricoIcon;
export const Star = NouricoIcon;
export const Tag = NouricoIcon;
export const Target = NouricoIcon;
export const Terminal = NouricoIcon;
export const Trash2 = NouricoIcon;
export const TriangleAlertIcon = NouricoIcon;
export const Trophy = NouricoIcon;
export const Upload = NouricoIcon;
export const UserPlus = NouricoIcon;
export const Users = NouricoIcon;
export const Wrench = NouricoIcon;
export const X = NouricoIcon;
export const XCircle = NouricoIcon;
export const XIcon = NouricoIcon;
export const Zap = NouricoIcon;
