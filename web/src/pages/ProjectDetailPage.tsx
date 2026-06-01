import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Typography, Descriptions, Statistic, Table, Button, Space, Tag, Card, Row, Col, Tabs, Input, message, Spin, Empty, Result } from "antd";
import { ArrowLeftOutlined, CopyOutlined, ReloadOutlined, DownloadOutlined } from "@ant-design/icons";

const { Title, Text } = Typography;

interface Project {
  id: number; name: string; description: string; data_category: string;
  capture_enabled: boolean; sample_rate: number; retention_days: number;
  has_export_token: boolean; created_at: string; updated_at: string;
}
interface Stats {
  project_id: number; total_captures: number; captures_today: number;
  total_request_size_bytes: number; total_response_size_bytes: number;
  routes_count: number; earliest_capture: string; latest_capture: string;
}
interface CaptureRecord {
  id: number; request_id: string; project_id: number;
  public_model_name: string; stream: boolean; status_code: number;
  body_skipped: boolean; file_path: string; request_size: number;
  response_size: number; created_at: string;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [project, setProject] = useState<Project | null>(null);
  const [stats, setStats] = useState<Stats | null>(null);
  const [captures, setCaptures] = useState<CaptureRecord[]>([]);
  const [captureTotal, setCaptureTotal] = useState(0);
  const [capLoading, setCapLoading] = useState(false);
  const [exportToken, setExportToken] = useState("");
  const [cursor, setCursor] = useState<number | null>(null);
  const [pageLoading, setPageLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const fetchProject = useCallback(async () => {
    try {
      const res = await fetch(`/api/projects/${id}`, { credentials: "include" });
      if (res.ok) setProject(await res.json());
      else setLoadError(`Failed to load project (HTTP ${res.status})`);
    } catch { setLoadError("Network error loading project"); }
    finally { setPageLoading(false); }
  }, [id]);

  const fetchStats = useCallback(async () => {
    const res = await fetch(`/api/projects/${id}/stats`, { credentials: "include" });
    if (res.ok) setStats(await res.json());
  }, [id]);

  const fetchCaptures = useCallback(async (c?: number) => {
    setCapLoading(true);
    try {
      const params = new URLSearchParams({ page_size: "50" });
      if (c) params.set("cursor", String(c));
      const res = await fetch(`/api/projects/${id}/captures?${params}`, { credentials: "include" });
      if (res.ok) {
        const data = await res.json();
        setCaptures(data.data || []);
        setCaptureTotal(data.total || 0);
        setCursor(data.next_cursor ?? null);
      }
    } finally { setCapLoading(false); }
  }, [id]);

  useEffect(() => { fetchProject(); fetchStats(); fetchCaptures(); }, [fetchProject, fetchStats, fetchCaptures]);

  const generateToken = async () => {
    const res = await fetch(`/api/projects/${id}/export-token`, { method: "POST", credentials: "include" });
    if (res.ok) {
      const data = await res.json();
      setExportToken(data.export_token);
      fetchProject();
    }
  };

  const copyToken = () => {
    navigator.clipboard.writeText(exportToken);
    message.success("Token copied to clipboard");
  };

  const captureColumns = [
    { title: "Request ID", dataIndex: "request_id", key: "rid",
      render: (v: string) => <span className="mono" style={{ fontSize: 12 }}>{v.slice(0, 12)}...</span> },
    { title: "Model", dataIndex: "public_model_name", key: "model",
      render: (v: string) => <span className="mono">{v}</span> },
    { title: "Stream", dataIndex: "stream", key: "stream",
      render: (v: boolean) => v ? <Tag color="blue">Stream</Tag> : <Tag>Non-stream</Tag> },
    { title: "Status", dataIndex: "status_code", key: "status",
      render: (v: number) => <Tag color={v >= 200 && v < 300 ? "green" : "red"}>{v}</Tag> },
    { title: "Req Size", dataIndex: "request_size", key: "req",
      render: (v: number) => formatBytes(v) },
    { title: "Resp Size", dataIndex: "response_size", key: "resp",
      render: (v: number) => formatBytes(v) },
    { title: "Captured", dataIndex: "created_at", key: "time",
      render: (v: string) => new Date(v).toLocaleString() },
  ];

  if (pageLoading) return <div style={{ padding: 24, textAlign: "center" }}><Spin size="large" tip="Loading project..." /></div>;
  if (loadError) return <div style={{ padding: 24 }}><Result status="error" title="Failed to Load" subTitle={loadError} extra={<Button onClick={() => navigate("/projects")}>Back to Projects</Button>} /></div>;
  if (!project) return <div style={{ padding: 24 }}><Result status="404" title="Project Not Found" extra={<Button onClick={() => navigate("/projects")}>Back to Projects</Button>} /></div>;

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate("/projects")}>Back</Button>
        <Title level={3} style={{ margin: 0 }}>{project.name}</Title>
        <Tag color={project.capture_enabled ? "green" : "default"}>
          {project.capture_enabled ? "Capturing" : "Paused"}
        </Tag>
      </Space>

      {project.description && <Text type="secondary" style={{ display: "block", marginBottom: 16 }}>{project.description}</Text>}

      {stats && (
        <Row gutter={16} style={{ marginBottom: 24 }}>
          <Col span={6}><Card size="small"><Statistic title="Total Captures" value={stats.total_captures} /></Card></Col>
          <Col span={6}><Card size="small"><Statistic title="Today" value={stats.captures_today} /></Card></Col>
          <Col span={6}><Card size="small"><Statistic title="Data Size" value={formatBytes(stats.total_request_size_bytes + stats.total_response_size_bytes)} /></Card></Col>
          <Col span={6}><Card size="small"><Statistic title="Routes" value={stats.routes_count} /></Card></Col>
        </Row>
      )}

      <Tabs items={[
        { key: "captures", label: "Captures", children: (
          <div>
            <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 16 }}>
              <Text type="secondary">Total: {captureTotal}</Text>
              <Space>
                <Button icon={<ReloadOutlined />} onClick={() => fetchCaptures()}>Refresh</Button>
                <Button icon={<DownloadOutlined />} href={`/api/projects/${id}/captures/export?download=true`}>
                  Export JSONL
                </Button>
              </Space>
            </div>
            <Table columns={captureColumns} dataSource={captures} rowKey="id" loading={capLoading}
              pagination={false} size="small"
              locale={{ emptyText: <Empty description={project.capture_enabled ? "No captures recorded yet. Send requests through associated routes." : "Capture is paused. Enable it in project settings."} /> }} />
            {cursor && (
              <div style={{ textAlign: "center", marginTop: 16 }}>
                <Button onClick={() => fetchCaptures(cursor)}>Load More</Button>
              </div>
            )}
          </div>
        )},
        { key: "settings", label: "Settings", children: (
          <div>
            <Descriptions column={2} bordered size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label="Category">{project.data_category}</Descriptions.Item>
              <Descriptions.Item label="Sample Rate">{(project.sample_rate * 100).toFixed(0)}%</Descriptions.Item>
              <Descriptions.Item label="Retention">{project.retention_days} days</Descriptions.Item>
              <Descriptions.Item label="Export Token">{project.has_export_token ? "Configured" : "Not configured"}</Descriptions.Item>
              <Descriptions.Item label="Created">{new Date(project.created_at).toLocaleString()}</Descriptions.Item>
              <Descriptions.Item label="Updated">{new Date(project.updated_at).toLocaleString()}</Descriptions.Item>
            </Descriptions>

            <Card title="Export Token" size="small">
              <Text type="secondary" style={{ display: "block", marginBottom: 12 }}>
                Generate a token for ipsa-eval to access capture exports via API.
              </Text>
              <Space>
                <Button onClick={generateToken} type="primary">
                  {project.has_export_token ? "Rotate Token" : "Generate Token"}
                </Button>
              </Space>
              {exportToken && (
                <div style={{ marginTop: 12 }}>
                  <Input.TextArea value={exportToken} readOnly rows={2} style={{ fontFamily: "monospace", fontSize: 12 }} />
                  <Button icon={<CopyOutlined />} onClick={copyToken} style={{ marginTop: 8 }} size="small">
                    Copy
                  </Button>
                </div>
              )}
            </Card>
          </div>
        )},
      ]} />
    </div>
  );
}

export default ProjectDetailPage;
