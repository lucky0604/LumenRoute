import { Tag } from "antd";
import type { TagProps } from "antd";

export type StatusLabel =
  | "Healthy"
  | "Enabled"
  | "All access"
  | "Scoped"
  | "Degraded"
  | "Unhealthy"
  | "Disabled"
  | "Unverified"
  | "Auth required"
  | "Auth failed"
  | "No traffic"
  | "Unknown"
  | "Configured";

const statusColorMap: Record<StatusLabel, TagProps["color"]> = {
  Healthy: "success",
  Enabled: "success",
  "All access": "processing",
  Scoped: "warning",
  Degraded: "warning",
  Unhealthy: "error",
  Disabled: "default",
  Unverified: "processing",
  "Auth required": "warning",
  "Auth failed": "error",
  "No traffic": "default",
  Unknown: "default",
  Configured: "success",
};

interface StatusChipProps {
  label: StatusLabel;
}

function StatusChip({ label }: StatusChipProps) {
  if (!label || label.length === 0) {
    return <Tag color="default">—</Tag>;
  }

  return <Tag color={statusColorMap[label]}>{label}</Tag>;
}

export default StatusChip;
