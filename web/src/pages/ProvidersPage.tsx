import { useState, useEffect, useCallback } from "react";
import { Typography, Table, Button, Modal, Form, Input, Select, Space, Popconfirm, App } from "antd";
import { PlusOutlined, CheckCircleOutlined } from "@ant-design/icons";
import StatusChip from "../components/StatusChip";
import EmptyState from "../components/EmptyState";
import type { StatusLabel } from "../components/StatusChip";

const { Title, Text } = Typography;

interface Provider {
  id: number; name: string; description: string; provider_type: string;
  engine: string; base_url: string; auth_mode: string; health_check_path: string;
  health_status: string; last_error: string; enabled: boolean; created_at: string;
}

function healthLabel(s: string): StatusLabel {
  if (s === "healthy") return "Healthy";
  if (s === "unhealthy") return "Unhealthy";
  return "Unknown";
}

function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [form] = Form.useForm();
  const { message } = App.useApp();

  const fetchProviders = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/providers", { credentials: "include" });
      if (res.status === 401) { setProviders([]); return; }
      const data = await res.json();
      setProviders(Array.isArray(data) ? data : []);
    } finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchProviders(); }, [fetchProviders]);

  const handleCreate = async () => {
    const values = await form.validateFields();
    setCreateLoading(true);
    try {
      const res = await fetch("/api/providers", {
        method: "POST", headers: { "Content-Type": "application/json" },
        credentials: "include", body: JSON.stringify({ ...values, enabled: true }),
      });
      if (res.status === 401) {
        message.error("Session expired. Please log in again.");
        return;
      }
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        message.error(body?.error || `Server error: ${res.status}`);
        return;
      }
      message.success("Provider created");
      setCreateOpen(false);
      form.resetFields();
      fetchProviders();
    } finally {
      setCreateLoading(false);
    }
  };

  const handleDelete = async (id: number) => {
    await fetch(`/api/providers/${id}`, { method: "DELETE", credentials: "include" });
    fetchProviders();
  };

  const handleCheck = async (id: number) => {
    await fetch(`/api/providers/${id}/check`, { method: "POST", credentials: "include" });
    fetchProviders();
  };

  const healthyCount = providers.filter(p => p.health_status === "healthy").length;
  const unhealthyCount = providers.filter(p => p.health_status === "unhealthy").length;
  const unknownCount = providers.length - healthyCount - unhealthyCount;

  const columns = [
    { title: "Name", dataIndex: "name", key: "name" },
    { title: "Engine", dataIndex: "engine", key: "engine" },
    { title: "Base URL", dataIndex: "base_url", key: "base_url", render: (v: string) => <span className="mono">{v}</span> },
    { title: "Health", dataIndex: "health_status", key: "health", render: (s: string) => <StatusChip label={healthLabel(s)} /> },
    { title: "Enabled", dataIndex: "enabled", key: "enabled", render: (v: boolean) => <StatusChip label={v ? "Enabled" : "Disabled"} /> },
    { title: "Created", dataIndex: "created_at", key: "created_at", render: (t: string) => t ? new Date(t).toLocaleString() : "-" },
    { title: "Actions", key: "actions", render: (_: unknown, r: Provider) => (
      <Space>
        <Button size="small" icon={<CheckCircleOutlined />} onClick={() => handleCheck(r.id)}>Check</Button>
        <Popconfirm title="Delete this provider permanently?" onConfirm={() => handleDelete(r.id)}>
          <Button size="small" danger>Delete</Button>
        </Popconfirm>
      </Space>
    )},
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 16 }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>Providers</Title>
          <Text type="secondary">Register and monitor upstream model providers.</Text>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>Add Provider</Button>
      </div>

      <Space style={{ marginBottom: 16 }}>
        <StatusChip label="Healthy" /> {healthyCount}
        <StatusChip label="Unhealthy" /> {unhealthyCount}
        <StatusChip label="Unknown" /> {unknownCount}
        <Text type="secondary">Total: {providers.length}</Text>
      </Space>

      <div className="table-wrapper">
        <Table
          columns={columns}
          dataSource={providers}
          rowKey="id"
          loading={loading}
          locale={{ emptyText: <EmptyState reason="No providers yet. Add your first upstream provider before creating routes." action="Add Provider" onAction={() => setCreateOpen(true)} /> }}
        />
      </div>

      <Modal title="Add Provider" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={handleCreate} okText="Create" confirmLoading={createLoading}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true, message: "Provider name is required" }]}>
            <Input placeholder="e.g. my-vllm-cluster" />
          </Form.Item>
          <Form.Item name="provider_type" label="Type" initialValue="openai">
            <Select options={[{ label: "OpenAI-compatible", value: "openai" }]} />
          </Form.Item>
          <Form.Item name="engine" label="Engine" initialValue="vllm">
            <Select options={[{ label: "vLLM", value: "vllm" }, { label: "SGLang", value: "sglang" }, { label: "Ollama", value: "ollama" }, { label: "OpenAI", value: "openai" }]} />
          </Form.Item>
          <Form.Item name="base_url" label="Base URL" rules={[{ required: true }]}>
            <Input placeholder="e.g. http://192.168.1.100:8000" />
          </Form.Item>
          <Form.Item name="auth_mode" label="Auth Mode" initialValue="none">
            <Select options={[{ label: "None", value: "none" }, { label: "API Key", value: "api_key" }]} />
          </Form.Item>
          <Form.Item name="health_check_path" label="Health Check Path">
            <Input placeholder="e.g. /health (default: /models)" />
          </Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default ProvidersPage;
