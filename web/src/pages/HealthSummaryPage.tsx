import { useState, useEffect } from "react";
import { Typography, Card, Table, Space, Button } from "antd";
import { BarChartOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import StatusChip from "../components/StatusChip";
import type { StatusLabel } from "../components/StatusChip";

const { Title, Text } = Typography;

interface Provider {
  id: number; name: string; health_status: string; last_error: string; last_check_at: string;
}

function healthLabel(s: string): StatusLabel {
  if (s === "healthy") return "Healthy";
  if (s === "unhealthy") return "Unhealthy";
  return "Unknown";
}

function HealthSummaryPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const navigate = useNavigate();

  useEffect(() => {
    fetch("/api/providers", { credentials: "include" })
      .then(r => r.ok ? r.json() : []).then(setProviders);
  }, []);

  const unhealthy = providers.filter(p => p.health_status === "unhealthy");
  const healthy = providers.filter(p => p.health_status === "healthy");

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>Health Summary</Title>
      <Text type="secondary" style={{ display: "block", marginBottom: 16 }}>Quick provider-health overview for routing operations.</Text>

      <Space orientation="vertical" size="middle" style={{ width: "100%" }}>
        <Card title="Overview" size="small">
          <Space>
            <StatusChip label="Healthy" /> {healthy.length}
            <StatusChip label="Unhealthy" /> {unhealthy.length}
            <Text type="secondary">Total: {providers.length}</Text>
            <Button type="link" icon={<BarChartOutlined />} onClick={() => navigate("/model-performance")}>
              Model Performance
            </Button>
          </Space>
        </Card>

        {unhealthy.length > 0 && (
          <Card title="Unhealthy Providers" size="small">
            <Table dataSource={unhealthy} rowKey="id" size="small" columns={[
              { title: "Name", dataIndex: "name" },
              { title: "Error", dataIndex: "last_error" },
              { title: "Last Check", dataIndex: "last_check_at", render: (t: string) => t ? new Date(t).toLocaleString() : "-" },
            ]} onRow={() => ({ onClick: () => navigate("/providers"), style: { cursor: "pointer" } })} />
          </Card>
        )}

        <Card title="All Providers" size="small">
          <Table dataSource={providers} rowKey="id" size="small" columns={[
            { title: "Name", dataIndex: "name" },
            { title: "Status", dataIndex: "health_status", render: (s: string) => <StatusChip label={healthLabel(s)} /> },
            { title: "Last Check", dataIndex: "last_check_at", render: (t: string) => t ? new Date(t).toLocaleString() : "-" },
          ]} onRow={() => ({ onClick: () => navigate("/providers"), style: { cursor: "pointer" } })} />
        </Card>
      </Space>
    </div>
  );
}

export default HealthSummaryPage;
