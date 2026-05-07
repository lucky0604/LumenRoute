import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Typography, Descriptions, Table, Button, Space, Tag, Alert, Segmented, Card, Row, Col, List, Statistic, message } from "antd";
import { ReloadOutlined, CopyOutlined, FileSearchOutlined } from "@ant-design/icons";

const { Title, Text } = Typography;

interface TargetFacts {
  id: number; route_id: number; route_name: string;
  public_model_name: string; upstream_model_name: string;
  provider_name: string; provider_base_url: string;
  provider_engine: string; provider_health: string;
  last_check_at: string; last_provider_error: string; enabled: boolean;
}

interface ModelSummary {
  public_model_name: string; upstream_model_name: string; provider_name: string;
  target_id: number; request_count: number; error_count: number; error_rate: number;
  avg_latency_ms: number; p95_latency_ms: number; total_tokens: number;
  stream_count: number; stream_completed_rate: number;
  last_error_code: string; last_seen_at: string;
}

interface RequestLogBrief {
  id: number; status_code: number; upstream_status_code: number;
  error_code: string; error_message: string; latency_ms: number;
  stream_completed: boolean | null; public_model_name: string; created_at: string;
}

interface TargetDiagnosisData {
  target: TargetFacts;
  summary: ModelSummary;
  recent_failures: RequestLogBrief[];
  slow_requests: RequestLogBrief[];
  operator_commands: { models_curl: string };
}

function healthTag(status: string) {
  if (status === "healthy") return <Tag color="green">Healthy</Tag>;
  if (status === "unhealthy") return <Tag color="red">Unhealthy</Tag>;
  if (status === "deleted") return <Tag color="red">Deleted</Tag>;
  return <Tag color="default">{status || "Unknown"}</Tag>;
}

function requestHealthTag(rate: number, count: number) {
  if (count === 0) return <Tag color="default">No traffic</Tag>;
  if (rate > 0.05) return <Tag color="red">Failing</Tag>;
  if (rate > 0.01) return <Tag color="orange">Degraded</Tag>;
  return <Tag color="green">Healthy</Tag>;
}

function TargetDiagnosisPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [data, setData] = useState<TargetDiagnosisData | null>(null);
  const [window, setWindow] = useState<string>("1h");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/diagnostics/targets/${id}?window=${window}`, { credentials: "include" });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.error || `HTTP ${res.status}`);
      }
      setData(await res.json());
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, [id, window]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const copyText = async (text: string, label: string) => {
    await navigator.clipboard.writeText(text);
    message.success(`${label} copied`);
  };

  if (error && !data) {
    return (
      <div style={{ padding: 24 }}>
        <Alert type="error" message="Failed to load target diagnosis" description={error} showIcon
          action={<Button size="small" onClick={fetchData}>Retry</Button>} />
      </div>
    );
  }

  const t = data?.target;
  const s = data?.summary;

  const failureCols = [
    { title: "Time", dataIndex: "created_at", key: "t", width: 150, render: (v: string) => new Date(v).toLocaleString() },
    { title: "Status", dataIndex: "upstream_status_code", key: "s", width: 70, render: (v: number) => <Tag color="red">{v}</Tag> },
    { title: "Error", dataIndex: "error_code", key: "e", ellipsis: true, render: (v: string) => v || "-" },
    { title: "Latency", dataIndex: "latency_ms", key: "l", width: 80, render: (v: number) => `${v}ms` },
  ];

  const slowCols = [
    { title: "Time", dataIndex: "created_at", key: "t", width: 150, render: (v: string) => new Date(v).toLocaleString() },
    { title: "Status", dataIndex: "status_code", key: "s", width: 70, render: (v: number) => <Tag color={v >= 400 ? "red" : "green"}>{v}</Tag> },
    { title: "Error", dataIndex: "error_code", key: "e", ellipsis: true, render: (v: string) => v ? <Tag color="red">{v}</Tag> : null },
    { title: "Latency", dataIndex: "latency_ms", key: "l", width: 80, render: (v: number) => `${v}ms` },
    { title: "Stream", dataIndex: "stream_completed", key: "sc", width: 80,
      render: (v: boolean | null) => v === null ? "-" : v ? <Tag color="green">Done</Tag> : <Tag color="red">Incomplete</Tag> },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16, justifyContent: "space-between", width: "100%" }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>
            {t?.public_model_name || "Loading..."} / Target #{id}
          </Title>
          <Text type="secondary">
            {t?.upstream_model_name} &middot; {t?.provider_name} &middot; {t?.provider_engine}
          </Text>
        </div>
        <Space>
          <Segmented options={["5m", "1h", "24h"]} value={window} onChange={(v) => setWindow(v as string)} />
          <Button icon={<ReloadOutlined />} onClick={fetchData}>Refresh</Button>
        </Space>
      </Space>

      {loading && !data ? (
        <Card loading style={{ marginBottom: 16 }} />
      ) : (
        <>
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col xs={12} sm={6}>
              <Card size="small"><Statistic title="Request Health"
                valueRender={() => requestHealthTag(s?.error_rate || 0, s?.request_count || 0)} /></Card>
            </Col>
            <Col xs={12} sm={6}>
              <Card size="small"><Statistic title="Provider Health"
                valueRender={() => healthTag(t?.provider_health || "unknown")} /></Card>
            </Col>
            <Col xs={12} sm={6}>
              <Card size="small"><Statistic title="Stream Done"
                value={s?.stream_count ? `${((s.stream_completed_rate || 0) * 100).toFixed(1)}%` : "-"} /></Card>
            </Col>
            <Col xs={12} sm={6}>
              <Card size="small"><Statistic title="P95 Latency"
                value={s?.request_count ? Math.round(s.p95_latency_ms || 0) : "-"} suffix={s?.request_count ? "ms" : ""} /></Card>
            </Col>
          </Row>

          {t && (
            <Card size="small" title="Routing Facts" style={{ marginBottom: 16 }}>
              <Descriptions column={{ xs: 1, sm: 2, md: 3 }} size="small" bordered>
                <Descriptions.Item label="Public Model">{t.public_model_name}</Descriptions.Item>
                <Descriptions.Item label="Upstream Model">{t.upstream_model_name}</Descriptions.Item>
                <Descriptions.Item label="Provider">{t.provider_name}</Descriptions.Item>
                <Descriptions.Item label="Base URL">
                  <Text copyable={{ text: t.provider_base_url }}>{t.provider_base_url}</Text>
                </Descriptions.Item>
                <Descriptions.Item label="Engine">{t.provider_engine}</Descriptions.Item>
                <Descriptions.Item label="Health">{healthTag(t.provider_health)}</Descriptions.Item>
                {t.last_check_at && <Descriptions.Item label="Last Check">{new Date(t.last_check_at).toLocaleString()}</Descriptions.Item>}
                {t.last_provider_error && <Descriptions.Item label="Last Error"><Text type="danger">{t.last_provider_error}</Text></Descriptions.Item>}
              </Descriptions>
              <Space style={{ marginTop: 12 }}>
                <Button size="small" icon={<CopyOutlined />} onClick={() => copyText(t.provider_base_url, "Provider URL")}>Copy Provider URL</Button>
                <Button size="small" icon={<CopyOutlined />}
                  onClick={() => copyText(data?.operator_commands?.models_curl || "", "Curl command")}>
                  Copy /v1/models Curl
                </Button>
                <Button size="small" icon={<FileSearchOutlined />}
                  onClick={() => navigate(`/request-logs?target_id=${id}`)}>
                  View Filtered Logs
                </Button>
              </Space>
            </Card>
          )}

          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col xs={24} lg={12}>
              <Card size="small" title="Recent Failures" style={{ height: "100%" }}>
                <Table columns={failureCols} dataSource={data?.recent_failures || []} rowKey="id"
                  size="small" pagination={false} scroll={{ x: 400 }}
                  locale={{ emptyText: "No failures recorded" }} />
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card size="small" title="Slow Requests" style={{ height: "100%" }}>
                <Table columns={slowCols} dataSource={data?.slow_requests || []} rowKey="id"
                  size="small" pagination={false} scroll={{ x: 400 }}
                  locale={{ emptyText: "No slow requests in this window" }} />
              </Card>
            </Col>
          </Row>

          <Card size="small" title="Operator Next Checks">
            <List
              size="small"
              dataSource={[
                `Verify upstream: ${data?.operator_commands?.models_curl || "N/A"}`,
                `Check GPU host with nvidia-smi for resource usage`,
                `Inspect service logs for ${t?.provider_name || "provider"}`,
              ]}
              renderItem={(item) => <List.Item>{item}</List.Item>}
            />
          </Card>
        </>
      )}
    </div>
  );
}

export default TargetDiagnosisPage;
