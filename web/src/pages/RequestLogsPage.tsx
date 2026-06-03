import { useState, useEffect, useCallback } from "react";
import { Typography, Table, Input, Select, Button, Space, Drawer, Descriptions } from "antd";
import { SearchOutlined, ReloadOutlined } from "@ant-design/icons";
import StatusChip from "../components/StatusChip";
import EmptyState from "../components/EmptyState";
import type { StatusLabel } from "../components/StatusChip";

const { Title, Text } = Typography;

interface Log {
  id: number; request_id: string; public_model_name: string; provider_name: string;
  status_code: number; latency_ms: number; stream: boolean; total_tokens: number | null;
  time_to_first_chunk_ms: number | null; stream_completed: boolean | null;
  error_code: string; error_message: string; upstream_model_name: string; client_ip: string; created_at: string;
  request_body: string; response_body: string;
}

function statusLabel(code: number, hasError: boolean): StatusLabel {
  if (code >= 500 || hasError) return "Unhealthy";
  if (code >= 400) return "Degraded";
  return "Healthy";
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
    { title: "Request ID", dataIndex: "request_id", key: "rid", width: 140, render: (v: string) => <span className="mono">{v?.slice(0, 16)}&hellip;</span> },
    { title: "Model", dataIndex: "public_model_name", key: "model", render: (v: string) => <span className="mono">{v}</span> },
    { title: "Status", dataIndex: "status_code", key: "status", width: 80,
      render: (v: number, r: Log) => <StatusChip label={statusLabel(v, !!r.error_code)} /> },
    { title: "Latency", dataIndex: "latency_ms", key: "lat", width: 80, render: (v: number) => `${v}ms` },
    { title: "Stream", dataIndex: "stream", key: "stream", width: 70, render: (v: boolean) => v ? "Yes" : "No" },
    { title: "Tokens", dataIndex: "total_tokens", key: "tokens", width: 70, render: (v: number | null) => v ?? "-" },
    { title: "Error", dataIndex: "error_code", key: "err", width: 80, render: (v: string) => v ? <StatusChip label="Unhealthy" /> : null },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 16, flexWrap: "wrap", gap: 8 }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>Request Logs</Title>
          <Text type="secondary">Inspect request evidence and trace errors.</Text>
        </div>
        <Button icon={<ReloadOutlined />} onClick={fetchLogs}>Refresh</Button>
      </div>

      <Space style={{ marginBottom: 16, flexWrap: "wrap" }}>
        <Input placeholder="Model" value={model} onChange={e => setModel(e.target.value)} style={{ width: 160 }} prefix={<SearchOutlined />} allowClear />
        <Input placeholder="Provider" value={provider} onChange={e => setProvider(e.target.value)} style={{ width: 160 }} allowClear />
        <Select value={statusFilter} onChange={setStatusFilter} style={{ width: 120 }} allowClear placeholder="Status">
          <Select.Option value="200">200</Select.Option>
          <Select.Option value="400">4xx</Select.Option>
          <Select.Option value="500">5xx</Select.Option>
        </Select>
        <Button type={errorOnly ? "primary" : "default"} onClick={() => setErrorOnly(!errorOnly)}>Errors Only</Button>
      </Space>

      <div className="table-wrapper">
        <Table
          columns={columns}
          dataSource={logs}
          rowKey="id"
          loading={loading}
          size="small"
          onRow={r => ({ onClick: () => setDetailLog(r), style: { cursor: "pointer" } })}
          locale={{ emptyText: <EmptyState reason="No logs in this window." /> }}
        />
      </div>

      <Drawer title="Request Detail" open={!!detailLog} onClose={() => setDetailLog(null)} width={600}>
        {detailLog && (
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="Request ID"><span className="mono">{detailLog.request_id}</span></Descriptions.Item>
            <Descriptions.Item label="Model"><span className="mono">{detailLog.public_model_name}</span></Descriptions.Item>
            <Descriptions.Item label="Upstream"><span className="mono">{detailLog.upstream_model_name}</span></Descriptions.Item>
            <Descriptions.Item label="Provider">{detailLog.provider_name}</Descriptions.Item>
            <Descriptions.Item label="Status">{detailLog.status_code}</Descriptions.Item>
            <Descriptions.Item label="Latency">{detailLog.latency_ms}ms</Descriptions.Item>
            {detailLog.stream && detailLog.time_to_first_chunk_ms !== null && (
              <Descriptions.Item label="TTFT">{detailLog.time_to_first_chunk_ms}ms</Descriptions.Item>
            )}
            <Descriptions.Item label="Stream">{detailLog.stream ? "Yes" : "No"}</Descriptions.Item>
            {detailLog.stream && detailLog.stream_completed !== null && (
              <Descriptions.Item label="Stream Completed">{detailLog.stream_completed ? "Yes" : "No (中断)"}</Descriptions.Item>
            )}
            <Descriptions.Item label="Tokens">{detailLog.total_tokens ?? "-"}</Descriptions.Item>
            <Descriptions.Item label="Client IP">{detailLog.client_ip}</Descriptions.Item>
            {detailLog.error_code && <Descriptions.Item label="Error">{detailLog.error_code}: {detailLog.error_message}</Descriptions.Item>}
            <Descriptions.Item label="Time">{new Date(detailLog.created_at).toLocaleString()}</Descriptions.Item>
            {detailLog.request_body && (
              <Descriptions.Item label="Request Body">
                <pre style={{ maxHeight: 200, overflow: "auto", fontSize: 12, whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
                  {detailLog.request_body}
                </pre>
              </Descriptions.Item>
            )}
            {detailLog.response_body && (
              <Descriptions.Item label="Response Body">
                <pre style={{ maxHeight: 200, overflow: "auto", fontSize: 12, whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
                  {detailLog.response_body}
                </pre>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
}

export default RequestLogsPage;
