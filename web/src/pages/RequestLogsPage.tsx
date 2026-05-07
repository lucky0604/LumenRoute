import { useState, useEffect, useCallback } from "react";
import { Typography, Table, Input, Select, Button, Space, Tag, Modal, Descriptions } from "antd";
import { SearchOutlined, ReloadOutlined } from "@ant-design/icons";

const { Title } = Typography;

interface Log {
  id: number; request_id: string; public_model_name: string; provider_name: string;
  status_code: number; latency_ms: number; stream: boolean; total_tokens: number | null;
  error_code: string; error_message: string; upstream_model_name: string; client_ip: string; created_at: string;
}

function RequestLogsPage() {
  const [logs, setLogs] = useState<Log[]>([]);
  const [loading, setLoading] = useState(false);
  const [model, setModel] = useState("");
  const [provider, setProvider] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [errorOnly, setErrorOnly] = useState(false);
  const [detailLog, setDetailLog] = useState<Log | null>(null);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (model) params.set("model", model);
      if (provider) params.set("provider", provider);
      if (statusFilter) params.set("status_code", statusFilter);
      if (errorOnly) params.set("error_only", "true");
      const res = await fetch(`/api/request-logs?${params}`, { credentials: "include" });
      if (res.ok) setLogs(await res.json());
    } finally { setLoading(false); }
  }, [model, provider, statusFilter, errorOnly]);

  useEffect(() => { fetchLogs(); }, [fetchLogs]);

  const columns = [
    { title: "Time", dataIndex: "created_at", key: "time", render: (t: string) => new Date(t).toLocaleString(), width: 160 },
    { title: "Request ID", dataIndex: "request_id", key: "rid", render: (v: string) => <code style={{ fontSize: 11 }}>{v?.slice(0, 16)}...</code> },
    { title: "Model", dataIndex: "public_model_name", key: "model" },
    { title: "Status", dataIndex: "status_code", key: "status",
      render: (v: number) => <Tag color={v >= 500 ? "red" : v >= 400 ? "orange" : "green"}>{v}</Tag> },
    { title: "Latency", dataIndex: "latency_ms", key: "lat", render: (v: number) => `${v}ms` },
    { title: "Stream", dataIndex: "stream", key: "stream", render: (v: boolean) => v ? <Tag>stream</Tag> : null },
    { title: "Tokens", dataIndex: "total_tokens", key: "tokens", render: (v: number | null) => v ?? "-" },
    { title: "Error", dataIndex: "error_code", key: "err", render: (v: string) => v ? <Tag color="red">{v}</Tag> : null },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16, flexWrap: "wrap" }}>
        <Title level={3} style={{ margin: 0 }}>Request Logs</Title>
        <Input placeholder="Model" value={model} onChange={e => setModel(e.target.value)} style={{ width: 160 }} prefix={<SearchOutlined />} allowClear />
        <Input placeholder="Provider" value={provider} onChange={e => setProvider(e.target.value)} style={{ width: 160 }} allowClear />
        <Select value={statusFilter} onChange={setStatusFilter} style={{ width: 120 }} allowClear placeholder="Status">
          <Select.Option value="200">200</Select.Option>
          <Select.Option value="400">4xx</Select.Option>
          <Select.Option value="500">5xx</Select.Option>
        </Select>
        <Button type={errorOnly ? "primary" : "default"} onClick={() => setErrorOnly(!errorOnly)}>Errors Only</Button>
        <Button icon={<ReloadOutlined />} onClick={fetchLogs}>Refresh</Button>
      </Space>
      <Table columns={columns} dataSource={logs} rowKey="id" loading={loading} size="small"
        onRow={record => ({ onClick: () => setDetailLog(record), style: { cursor: "pointer" } })} />

      <Modal title="Request Detail" open={!!detailLog} onCancel={() => setDetailLog(null)} footer={null} width={600}>
        {detailLog && <Descriptions column={1} size="small" bordered>
          <Descriptions.Item label="Request ID"><code>{detailLog.request_id}</code></Descriptions.Item>
          <Descriptions.Item label="Model">{detailLog.public_model_name}</Descriptions.Item>
          <Descriptions.Item label="Upstream">{detailLog.upstream_model_name}</Descriptions.Item>
          <Descriptions.Item label="Provider">{detailLog.provider_name}</Descriptions.Item>
          <Descriptions.Item label="Status">{detailLog.status_code}</Descriptions.Item>
          <Descriptions.Item label="Latency">{detailLog.latency_ms}ms</Descriptions.Item>
          <Descriptions.Item label="Stream">{detailLog.stream ? "Yes" : "No"}</Descriptions.Item>
          <Descriptions.Item label="Tokens">{detailLog.total_tokens ?? "-"}</Descriptions.Item>
          <Descriptions.Item label="Client IP">{detailLog.client_ip}</Descriptions.Item>
          {detailLog.error_code && <Descriptions.Item label="Error">{detailLog.error_code}: {detailLog.error_message}</Descriptions.Item>}
          <Descriptions.Item label="Time">{new Date(detailLog.created_at).toLocaleString()}</Descriptions.Item>
        </Descriptions>}
      </Modal>
    </div>
  );
}

export default RequestLogsPage;
