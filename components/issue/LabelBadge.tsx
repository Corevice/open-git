type LabelBadgeProps = {
  name: string;
  color: string;
};

function getTextColor(hex: string): string {
  const r = parseInt(hex.slice(0, 2), 16) / 255;
  const g = parseInt(hex.slice(2, 4), 16) / 255;
  const b = parseInt(hex.slice(4, 6), 16) / 255;
  const luminance = 0.2126 * r + 0.7152 * g + 0.0722 * b;
  return luminance > 0.5 ? "#000" : "#fff";
}

export function LabelBadge({ name, color }: LabelBadgeProps) {
  return (
    <span
      className="inline-block rounded-full px-2 py-0.5 text-[11px] font-semibold leading-[18px]"
      style={{
        backgroundColor: `#${color}`,
        color: getTextColor(color),
      }}
    >
      {name}
    </span>
  );
}
