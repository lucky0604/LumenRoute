import { Typography, Button } from "antd";

const { Text } = Typography;

interface EmptyStateProps {
  reason: string;
  action?: string;
  onAction?: () => void;
  filterCaused?: boolean;
  compact?: boolean;
}

function EmptyState({
  reason,
  action,
  onAction,
  filterCaused = false,
  compact = false,
}: EmptyStateProps) {
  const prefix = filterCaused ? "No results match current filters. " : "";

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        padding: compact ? "32px 16px" : "64px 24px",
        textAlign: "center",
      }}
    >
      <Text
        type="secondary"
        style={{ maxWidth: 400, fontSize: compact ? 13 : 14 }}
      >
        {prefix}
        {reason}
      </Text>
      {action && onAction && (
        <Button
          type="primary"
          ghost
          onClick={onAction}
          style={{ marginTop: compact ? 12 : 16 }}
        >
          {action}
        </Button>
      )}
    </div>
  );
}

export default EmptyState;
