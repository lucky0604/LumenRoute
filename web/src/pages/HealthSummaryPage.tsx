import { useState, useEffect } from "react";
import { Typography, Card, Tag, Table, Space, Button } from "antd";
import { BarChartOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";

const { Title } = Typography;

interface Provider {
  id: number; name: string; health_status: string; last_error: string; last_check_at: string;
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

      <Space orientation="vertical" size="middle" style={{ width: "100%" }}>
        <Card title="Overview" size="small">
          <Space>
            <Tag color="green">{healthy.length} Healthy</Tag>
            <Tag color="red">{unhealthy.length} Unhealthy</Tag>
            <Tag>{providers.length} Total</Tag>
            <Button type="link" icon={<BarChartOutlined />} onClick={() => navigate("/model-performance")}>
              Model Performance
            </Button>
          </Space>
        </Card>

        {unhealthy.length > 0 && <Card title="Unhealthy Providers" size="small">
          <Table dataSource={unhealthy} rowKey="id" columns={[
            { title: "Name", dataIndex: "name", key: "name" },
            { title: "Error", dataIndex: "last_error", key: "error" },
            { title: "Last Check", dataIndex: "last_check_at", key: "time", render: (t: string) => t ? new Date(t).toLocaleString() : "-" },
          ]} onRow={() => ({ onClick: () => navigate("/providers"), style: { cursor: "pointer" } })} size="small" />
        </Card>}

        <Card title="All Providers" size="small">
          <Table dataSource={providers} rowKey="id" columns={[
            { title: "Name", dataIndex: "name", key: "name" },
            { title: "Status", dataIndex: "health_status", key: "status",
              render: (s: string) => <Tag color={s === "healthy" ? "green" : s === "unhealthy" ? "red" : "default"}>{s}</Tag> },
            { title: "Last Check", dataIndex: "last_check_at", key: "time", render: (t: string) => t ? new Date(t).toLocaleString() : "-" },
          ]} onRow={() => ({ onClick: () => navigate("/providers"), style: { cursor: "pointer" } })} size="small" />
        </Card>
      </Space>
    </div>
  );
}

export default HealthSummaryPage;
