import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Typography, Table, Button, Space, Tag, Alert, Segmented, Statistic, Card } from "antd";
import { ReloadOutlined, WarningOutlined } from "@ant-design/icons";

const { Title } = Typography;

interface ModelSummary {
  public_model_name: string;
  upstream_model_name: string;
  provider_name: string;
  target_id: number;
  request_count: number;
  error_count: number;
  error_rate: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  total_tokens: number;
  stream_count: number;
  stream_completed_rate: number;
  last_error_code: string;
  last_seen_at: string;
}

interface OverviewResponse {
  window: string;
  models: ModelSummary[];
}

function severityTag(m: ModelSummary) {
  if (m.request_count === 0) return <Tag color="default">No traffic</Tag>;
  if (m.error_rate > 0.05 || (m.last_error_code && m.last_error_code !== "")) {
    return <Tag color="red">Failing</Tag>;
  }
  if (m.avg_latency_ms > 10000 || m.stream_completed_rate < 0.95) {
    return <Tag color="orange">Degraded</Tag>;
  }
  return <Tag color="green">Healthy</Tag>;
}

function severityRank(m: ModelSummary): number {
  if (m.request_count === 0) return 0;
  if (m.error_rate > 0.05 || (m.last_error_code && m.last_error_code !== "")) return 3;
  if (m.avg_latency_ms > 10000 || m.stream_completed_rate < 0.95) return 2;
  return 1;
}

function ModelPerformancePage() {
  const [models, setModels] = useState<ModelSummary[]>([]);
  const [window, setWindow] = useState<string>("1h");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/diagnostics/models?window=${window}`, { credentials: "include" });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.error || `HTTP ${res.status}`);
      }
      const data: OverviewResponse = await res.json();
      setModels(data.models || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, [window]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const sortedModels = [...models].sort((a, b) => {
    const sa = severityRank(a), sb = severityRank(b);
    if (sb !== sa) return sb - sa;
    if (b.error_count !== a.error_count) return b.error_count - a.error_count;
    return b.p95_latency_ms - a.p95_latency_ms;
  });

  const totalRequests = models.reduce((s, m) => s + m.request_count, 0);
  const totalErrors = models.reduce((s, m) => s + m.error_count, 0);
  const worstP95 = models.reduce((max, m) => Math.max(max, m.p95_latency_ms), 0);
  const failingCount = models.filter(m => severityRank(m) === 3).length;
  const incompleteStreams = models.reduce(
    (s, m) => s + Math.round(m.stream_count * (1 - m.stream_completed_rate)), 0
  );

  const columns = [
    {
      title: "Severity", key: "severity", width: 90,
      sorter: (a: ModelSummary, b: ModelSummary) => severityRank(b) - severityRank(a),
      render: (_: unknown, r: ModelSummary) => severityTag(r),
    },
    {
      title: "Public Model", dataIndex: "public_model_name", key: "pub", ellipsis: true, width: 160,
    },
    {
      title: "Upstream Model", dataIndex: "upstream_model_name", key: "up", ellipsis: true, width: 160,
    },
    {
      title: "Provider", dataIndex: "provider_name", key: "prov", ellipsis: true, width: 180,
    },
    {
      title: "Target", dataIndex: "target_id", key: "tid", width: 70,
    },
    {
      title: "Reqs", dataIndex: "request_count", key: "reqs", width: 70,
      sorter: (a: ModelSummary, b: ModelSummary) => a.request_count - b.request_count,
    },
    {
      title: "Err%", key: "errpct", width: 80,
      sorter: (a: ModelSummary, b: ModelSummary) => a.error_rate - b.error_rate,
      render: (_: unknown, r: ModelSummary) =>
        r.request_count > 0 ? `${(r.error_rate * 100).toFixed(1)}%` : "-",
    },
    {
      title: "Avg (ms)", key: "avg", width: 90,
      sorter: (a: ModelSummary, b: ModelSummary) => a.avg_latency_ms - b.avg_latency_ms,
      render: (_: unknown, r: ModelSummary) =>
        r.request_count > 0 ? Math.round(r.avg_latency_ms).toLocaleString() : "-",
    },
    {
      title: "P95 (ms)", key: "p95", width: 90,
      sorter: (a: ModelSummary, b: ModelSummary) => a.p95_latency_ms - b.p95_latency_ms,
      render: (_: unknown, r: ModelSummary) =>
        r.request_count > 0 ? Math.round(r.p95_latency_ms).toLocaleString() : "-",
    },
    {
      title: "Stream Done", key: "stream", width: 100,
      render: (_: unknown, r: ModelSummary) =>
        r.stream_count > 0 ? `${(r.stream_completed_rate * 100).toFixed(1)}%` : "-",
    },
    {
      title: "Last Error", dataIndex: "last_error_code", key: "lerr", ellipsis: true, width: 140,
      render: (v: string) => v ? <Tag color="red">{v}</Tag> : null,
    },
    {
      title: "Last Seen", dataIndex: "last_seen_at", key: "ls", width: 150,
      render: (v: string) => v ? new Date(v).toLocaleString() : "-",
    },
    {
      title: "", key: "act", width: 90,
      render: (_: unknown, r: ModelSummary) => (
        <Button size="small" onClick={() => navigate(`/diagnostics/targets/${r.target_id}`)}>
          Diagnose
        </Button>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16, justifyContent: "space-between", width: "100%" }}>
        <Title level={3} style={{ margin: 0 }}>Model Performance</Title>
        <Space>
          <Segmented
            options={["5m", "1h", "24h"]}
            value={window}
            onChange={(v) => setWindow(v as string)}
          />
          <Button icon={<ReloadOutlined />} onClick={fetchData}>Refresh</Button>
        </Space>
      </Space>

      {error && (
        <Alert
          type="error"
          message="Failed to load model performance data"
          description={error}
          showIcon
          action={<Button size="small" onClick={fetchData}>Retry</Button>}
          style={{ marginBottom: 16 }}
        />
      )}

      <Space size="middle" style={{ marginBottom: 16, flexWrap: "wrap" }}>
        <Card size="small" style={{ minWidth: 120 }}>
          <Statistic title="Requests" value={totalRequests} />
        </Card>
        <Card size="small" style={{ minWidth: 120 }}>
          <Statistic title="Errors" value={totalErrors} valueStyle={totalErrors > 0 ? { color: "#EF4444" } : undefined} />
        </Card>
        <Card size="small" style={{ minWidth: 120 }}>
          <Statistic title="Worst P95" value={worstP95} suffix="ms" />
        </Card>
        <Card size="small" style={{ minWidth: 120 }}>
          <Statistic title="Failing" value={failingCount} prefix={failingCount > 0 ? <WarningOutlined /> : undefined} valueStyle={failingCount > 0 ? { color: "#EF4444" } : undefined} />
        </Card>
        <Card size="small" style={{ minWidth: 120 }}>
          <Statistic title="Incomplete Streams" value={incompleteStreams} />
        </Card>
      </Space>

      <Table
        columns={columns}
        dataSource={sortedModels}
        rowKey={(r) => `${r.target_id}-${r.public_model_name}`}
        loading={loading}
        size="small"
        scroll={{ x: 1400 }}
        pagination={false}
        locale={{ emptyText: "No request logs in this window" }}
      />
    </div>
  );
}

export default ModelPerformancePage;
